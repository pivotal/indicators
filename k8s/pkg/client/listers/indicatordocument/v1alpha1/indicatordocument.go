// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/pivotal/monitoring-indicator-protocol/k8s/pkg/apis/indicatordocument/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// IndicatorDocumentLister helps list IndicatorDocuments.
type IndicatorDocumentLister interface {
	// List lists all IndicatorDocuments in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.IndicatorDocument, err error)
	// IndicatorDocuments returns an object that can list and get IndicatorDocuments.
	IndicatorDocuments(namespace string) IndicatorDocumentNamespaceLister
	IndicatorDocumentListerExpansion
}

// indicatorDocumentLister implements the IndicatorDocumentLister interface.
type indicatorDocumentLister struct {
	indexer cache.Indexer
}

// NewIndicatorDocumentLister returns a new IndicatorDocumentLister.
func NewIndicatorDocumentLister(indexer cache.Indexer) IndicatorDocumentLister {
	return &indicatorDocumentLister{indexer: indexer}
}

// List lists all IndicatorDocuments in the indexer.
func (s *indicatorDocumentLister) List(selector labels.Selector) (ret []*v1alpha1.IndicatorDocument, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.IndicatorDocument))
	})
	return ret, err
}

// IndicatorDocuments returns an object that can list and get IndicatorDocuments.
func (s *indicatorDocumentLister) IndicatorDocuments(namespace string) IndicatorDocumentNamespaceLister {
	return indicatorDocumentNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// IndicatorDocumentNamespaceLister helps list and get IndicatorDocuments.
type IndicatorDocumentNamespaceLister interface {
	// List lists all IndicatorDocuments in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.IndicatorDocument, err error)
	// Get retrieves the IndicatorDocument from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.IndicatorDocument, error)
	IndicatorDocumentNamespaceListerExpansion
}

// indicatorDocumentNamespaceLister implements the IndicatorDocumentNamespaceLister
// interface.
type indicatorDocumentNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all IndicatorDocuments in the indexer for a given namespace.
func (s indicatorDocumentNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.IndicatorDocument, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.IndicatorDocument))
	})
	return ret, err
}

// Get retrieves the IndicatorDocument from the indexer for a given namespace and name.
func (s indicatorDocumentNamespaceLister) Get(name string) (*v1alpha1.IndicatorDocument, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("indicatordocument"), name)
	}
	return obj.(*v1alpha1.IndicatorDocument), nil
}
