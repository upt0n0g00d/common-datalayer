package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	layer "github.com/mimiro-io/common-datalayer"
)

// EnrichConfig is a function that can be used to enrich the config by reading additional files or environment variables
func EnrichConfig(config *layer.Config) error {
	//config.ApplicationConfig["env"] = "local"
	return nil
}

/*********************************************************************************************************************/

// SampleDataLayer is a sample implementation of the DataLayer interface
type SampleDataLayer struct {
	config   *layer.Config
	logger   layer.Logger
	metrics  layer.Metrics
	datasets map[string]*SampleDataset
}

func (dl *SampleDataLayer) Dataset(dataset string) (layer.Dataset, layer.LayerError) {
	ds, found := dl.datasets[dataset]
	if found {
		return ds, nil
	}
	return nil, layer.Errorf(layer.LayerErrorBadParameter, "dataset %s not found", dataset)
}

func (dl *SampleDataLayer) DatasetNames() []string {
	// create a slice of strings to hold the dataset names
	var datasetNames []string

	// add dataset names from the map to the slice
	for key := range dl.datasets {
		datasetNames = append(datasetNames, key)
	}
	return datasetNames
}

// no shutdown required
func (dl *SampleDataLayer) Stop(_ context.Context) error { return nil }

// NewSampleDataLayer is a factory function that creates a new instance of the sample data layer
// In this example we use it to populate the sample dataset with some data
func NewSampleDataLayer(conf *layer.Config, logger layer.Logger, metrics layer.Metrics) (layer.DataLayerService, error) {
	sampleDataLayer := &SampleDataLayer{config: conf, logger: logger, metrics: metrics}

	// initialize the datasets
	sampleDataLayer.datasets = make(map[string]*SampleDataset)

	// create a sample dataset
	sampleDataLayer.datasets["sample"] = &SampleDataset{dsName: "sample"}
	// loop to create 20 objects
	for i := 0; i < 20; i++ {
		// create a data object
		dataObject := DataObject{ID: "ID" + strconv.Itoa(i), Props: make(map[string]any)}

		// add some properties to the data object
		dataObject.Props["name"] = "name" + strconv.Itoa(i)
		dataObject.Props["description"] = "description" + strconv.Itoa(i)

		// add the data object to the sample dataset
		sampleDataLayer.datasets["sample"].data = append(sampleDataLayer.datasets["sample"].data, dataObject.AsBytes())
	}
	logger.Info(fmt.Sprintf("Initialized sample layer with %v objects", len(sampleDataLayer.datasets["sample"].data)))
	err := sampleDataLayer.UpdateConfiguration(conf)
	if err != nil {
		return nil, err
	}
	return sampleDataLayer, nil
}

// Initialize is called by the core service when the configuration is loaded.
// can be called many times if the configuration is reloaded
func (dl *SampleDataLayer) UpdateConfiguration(config *layer.Config) layer.LayerError {
	// just update mappings in this sample. no new dataset definitions are expected
	for k, v := range dl.datasets {
		for _, dsd := range config.DatasetDefinitions {
			if k == dsd.DatasetName {
				v.mappings = dsd.Mappings
			}
		}
	}
	return nil
}

/*********************************************************************************************************************/

// SampleDataset is a sample implementation of the Dataset interface, it provides a simple in-memory dataset in this case
type SampleDataset struct {
	dsName   string
	mappings []*layer.EntityPropertyMapping
	data     [][]byte
}

func (ds *SampleDataset) Write(item layer.Item) layer.LayerError {
	do := &DataObject{}
	if item.GetValue("id") != nil {
		do.ID = item.GetValue("id").(string)
	}
	do.Props = item.GetRaw()
	ds.data = append(ds.data, do.AsBytes())
	return nil
}

func (ds *SampleDataset) Name() string {
	return ds.dsName
}

// GetChanges returns an iterator over the changes since the given timestamp,
// The implementation uses the provided MappingEntityIterator and a custom DataObjectIterator
// to map the data objects to entities
func (ds *SampleDataset) Changes(since string, take int, _ bool) (layer.EntityIterator, layer.LayerError) {
	data := ds.data
	entityIterator := layer.NewMappingEntityIterator(
		ds.mappings,
		NewDataObjectIterator(data),
		nil)
	return entityIterator, nil
}

func (ds *SampleDataset) Entities(since string, take int) (layer.EntityIterator, layer.LayerError) {
	return ds.Changes(since, take, true)
}

func (ds *SampleDataset) BeginFullSync() layer.LayerError {
	return nil
}

func (ds *SampleDataset) CompleteFullSync() layer.LayerError {
	return nil
}

func (ds *SampleDataset) CancelFullSync() layer.LayerError {
	return nil
}

func (ds *SampleDataset) MetaData() map[string]any {
	return nil
}

/*********************************************************************************************************************/

// DataObject is the row/item type for the sample data layer. it implements the Item interface
// In addition to the Item interface, it also has a dedicated ID field and AsBytes,
// which is used to serialize the item for this specific layer
type DataObject struct {
	ID    string
	Props map[string]any
}

func (d *DataObject) AsBytes() []byte {
	b, _ := json.Marshal(d)
	return b
}

/*********************************************************************************************************************/

// DataObjectIterator is a sample implementation of the ItemIterator interface
// This is the glue between the data objects and the entity mapping
type DataObjectIterator struct {
	objects [][]byte
	pos     int
}

func (doi *DataObjectIterator) Token() string {
	//TODO implement me
	panic("implement me")
}

func (doi *DataObjectIterator) Close() {
	//TODO implement me
	panic("implement me")
}

func NewDataObjectIterator(objects [][]byte) *DataObjectIterator {
	doi := &DataObjectIterator{}
	doi.objects = objects
	doi.pos = 0
	return doi
}

func (doi *DataObjectIterator) Next() layer.Item {
	if doi.pos >= len(doi.objects) {
		return nil
	}
	b := doi.objects[doi.pos]
	doi.pos++
	obj := DataObject{}
	err := json.Unmarshal(b, &obj)
	if err != nil {
		panic(err)
	}
	res := &layer.DataItem{}
	res.PutRaw(obj.Props)
	res.SetValue("id", obj.ID)
	return res
}