package indicator

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"reflect"

	"github.com/cppforlife/go-patch/patch"
	"gopkg.in/yaml.v2"
)

func ApplyPatches(patches []Patch, documentBytes []byte) ([]byte, error) {
	var document interface{}
	err := yaml.Unmarshal(documentBytes, &document)
	if err != nil {
		return []byte{}, errors.New("failed to unmarshal document for patching")
	}

	for _, p := range patches {
		if MatchDocument(p.Match, documentBytes) {
			ops, err := patch.NewOpsFromDefinitions(p.Operations)
			if err != nil {
				log.Print(errors.New("failed to parse patch operations"))
				continue
			}
			for _, o := range ops {
				var tempDocument interface{}
				tempDocument, err = o.Apply(document)
				if err != nil {
					log.Print(errors.New("failed to apply patch operation"))
					continue
				}
				document = tempDocument
			}
		}
	}

	patched, err := yaml.Marshal(document)
	if err != nil {
		return []byte{}, errors.New("failed to marshal patch document")
	}
	return patched, nil
}

func MatchDocument(criteria Match, documentBytes []byte) bool {
	reader := ioutil.NopCloser(bytes.NewReader(documentBytes))
	product, err := ProductFromYAML(reader)
	if err != nil {
		return false
	}

	if criteria.Name != nil && *criteria.Name != product.Name {
		return false
	}
	if criteria.Version != nil && *criteria.Version != product.Version {
		return false
	}

	if criteria.Metadata != nil {
		reader := ioutil.NopCloser(bytes.NewReader(documentBytes))
		metadata, err := MetadataFromYAML(reader)
		if err != nil {
			return false
		}
		if !reflect.DeepEqual(metadata, criteria.Metadata) {
			return false
		}
	}

	return true
}