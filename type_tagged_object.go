package main

type TaggedObject struct {
	ObjectRef *TaggedObjectRef `json:"objectRef"`
	Tags      []string         `json:"tags"`
}

type TaggedObjectRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}
