package configuration

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	glob2 "github.com/gobwas/glob"
	"github.com/pivotal/monitoring-indicator-protocol/pkg/api_versions"
	v1 "github.com/pivotal/monitoring-indicator-protocol/pkg/k8s/apis/indicatordocument/v1"
	"github.com/pivotal/monitoring-indicator-protocol/pkg/kinds"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/yaml.v2"

	"github.com/pivotal/monitoring-indicator-protocol/pkg/indicator"
	"github.com/pivotal/monitoring-indicator-protocol/pkg/registry"
)

type RepositoryGetter func(Source) (*git.Repository, error)

func ParseSourcesFile(filePath string) ([]Source, error) {
	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, errors.New("could not parse sources file, error reading configuration file")
	}

	var f SourcesFile
	err = yaml.Unmarshal(fileBytes, &f)
	if err != nil {
		return nil, errors.New("could not parse sources file, error parsing configuration file yaml")
	}

	if err := Validate(f); err != nil {
		return nil, fmt.Errorf("could not parse sources file, configuration is not valid: %s", err)
	}

	return f.Sources, nil
}

func Read(sources []Source, repositoryGetter RepositoryGetter) ([]registry.PatchList, []v1.IndicatorDocument) {
	var patches []registry.PatchList
	var documents []v1.IndicatorDocument
	for _, source := range sources {

		switch source.Type {
		case "local":
			patch, err := indicator.ReadPatchFile(source.Path)
			if err != nil {
				log.Printf("failed to read local patch: %s", err)
				continue
			}

			patches = append(patches, registry.PatchList{
				Source:  source.Path,
				Patches: []indicator.Patch{patch},
			})
			log.Printf("Parsed %d patches from local sources", len(patches))
		case "git":
			repository, err := repositoryGetter(source)
			if err != nil {
				log.Print("failed to initialize git repository")
				continue
			}
			gitPatches, gitDocuments, err := parseRepositoryHead(source, repository)
			if err != nil {
				log.Print("failed to read patches in repository")
				continue
			}
			patches = append(patches, registry.PatchList{
				Source:  source.Repository,
				Patches: gitPatches,
			})
			documents = append(documents, gitDocuments...)
			log.Printf("Parsed %d documents and %d patches from git source", len(gitDocuments), len(gitPatches))
		default:
			log.Print("invalid source type, must be either \"local\" or \"git\"")
			continue
		}
	}

	return patches, documents
}

func Validate(f SourcesFile) error {
	for _, s := range f.Sources {
		switch s.Type {
		case "local":
			if s.Path == "" {
				return fmt.Errorf("path is required for local sources")
			}
		case "git":
			if s.Repository == "" {
				return fmt.Errorf("repository is required for git sources")
			}

			if s.Token != "" && !strings.Contains(s.Repository, "https://") {
				return fmt.Errorf("personal access token can only be used over HTTPS")
			}
		}
	}

	return nil
}

type SourcesFile struct {
	Sources []Source `yaml:"sources"`
}

type Source struct {
	Type       string `yaml:"type"`
	Path       string `yaml:"path"`
	Repository string `yaml:"repository"`
	Token      string `yaml:"token"`
	Key        string `yaml:"private_key"`
	Glob       string `yaml:"glob"`
}

func parseRepositoryHead(s Source, r *git.Repository) ([]indicator.Patch, []v1.IndicatorDocument, error) {
	ref, err := r.Head()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch repo head: %s\n", err)
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch commit: %s\n", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch commit tree: %s\n", err)
	}

	return retrievePatchesAndDocuments(tree.Files(), s.Glob)
}

// The error returned is always nil, but keeping this signature allows us to return it directly in parseRepositoryHead
func retrievePatchesAndDocuments(files *object.FileIter, glob string) ([]indicator.Patch, []v1.IndicatorDocument, error) {
	var patchesBytes []unparsedPatch
	var documentsBytes [][]byte

	if glob == "" {
		glob = "*.y*ml"
	}
	g := glob2.MustCompile(glob)

	err := files.ForEach(func(f *object.File) error {
		if g.Match(f.Name) {
			contents, err := f.Contents()
			if err != nil {
				log.Print("Error reading contents of patch or document file")
				return nil
			}

			version, err := indicator.ApiVersionFromYAML([]byte(contents))
			if err != nil {
				log.Print("Failed to parse file, perhaps you are missing `apiVersion` or your yaml is malformed?")
				return nil
			}

			switch version {
			case api_versions.V1:
				kind, err := indicator.KindFromYAML([]byte(contents))
				if err != nil {
					log.Print("Could not get the `kind` of the document, ensure that this key is present.")
					return nil
				}

				if kind == kinds.IndicatorDocument {
					documentsBytes = append(documentsBytes, []byte(contents))
				} else if kind == kinds.Patch {
					patchesBytes = append(patchesBytes, unparsedPatch{[]byte(contents), f.Name})
				} else {
					log.Printf("Invalid `kind` specified, must be `%s` or `%s`", kinds.IndicatorDocument, kinds.Patch)
				}
			}
		}
		return nil
	})

	patches := readPatches(patchesBytes)
	documents := processDocuments(documentsBytes, patches)
	return patches, documents, err
}

type unparsedPatch struct {
	YAMLBytes []byte
	Filename  string
}

func readPatches(unparsedPatches []unparsedPatch) []indicator.Patch {
	var patches []indicator.Patch
	for _, p := range unparsedPatches {
		reader := ioutil.NopCloser(bytes.NewReader(p.YAMLBytes))
		p, err := indicator.PatchFromYAML(reader)
		if err != nil {
			log.Println(err)
			continue
		}
		patches = append(patches, p)
	}
	return patches
}

func processDocuments(documentsBytes [][]byte, patches []indicator.Patch) []v1.IndicatorDocument {
	var documents []v1.IndicatorDocument
	for _, documentBytes := range documentsBytes {
		doc, errs := indicator.ProcessDocument(patches, documentBytes)
		if len(errs) > 0 {
			continue
		}

		documents = append(documents, doc)
	}
	return documents
}

