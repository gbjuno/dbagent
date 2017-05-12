package main

type Asserts struct {
	Regular   int `json:"regular"`
	Warning   int `json:"warning"`
	Msg       int `json:"msg"`
	User      int `json:"user"`
	Rollovers int `json:"rollovers"`
}

type Connections struct {
	Current      int `json:"current"`
	Available    int `json:"available"`
	TotalCreated int `json:"totalCreated"`
}

type Extra_info struct {
	Note             string `json:"note"`
	Heap_usage_bytes int    `json:"heap_usage_bytes"`
	Page_faults      int    `json:"page_faults"`
}

type GLock struct {
	Total   int `json:"total"`
	Readers int `json:"readers"`
	Writers int `json:"writers"`
}

type GlobalLock struct {
	TotalTime    int   `json:"totalTime"`
	CurrentQueue GLock `json:"currentQueue"`
	ActiveQueue  GLock `json:"activeQueue"`
}

type Lock struct {
	SR int `json:"r,omitempty"`
	SW int `json:"w,omitempty"`
	BR int `json:"R,omitempty"`
	BW int `json:"W,omitempty"`
}

type locks struct {
	GlobalLock map[string]Lock `json:"Global"`
	Database   map[string]Lock `json:"Database"`
	Collection map[string]Lock `json:"Collection"`
	Metadata   map[string]Lock `json:"MetaData"`
	Oplog      map[string]Lock `json:"oplog"`
}

type Network struct {
	BytesIn    int `json:"bytesIn"`
	BytesOut   int `json:"bytesOut"`
	NumRequest int `json:"numRequests"`
}

type Opcounters struct {
	Insert  int `json:"insert"`
	Query   int `json:"query"`
	Update  int `json:"update"`
	Delete  int `json:"delete"`
	Getmore int `json:"getmore"`
	Command int `json:"command"`
}

type OpcountersRepl struct {
	Insert  int `json:"insert"`
	Query   int `json:"query"`
	Update  int `json:"update"`
	Delete  int `json:"delete"`
	Getmore int `json:"getmore"`
	Command int `json:"command"`
}
