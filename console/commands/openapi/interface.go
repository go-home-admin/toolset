package openapi

type Spec struct {
	Swagger       string                `json:"swagger"`
	Info          Info                  `json:"info"`
	Host          string                `json:"host,omitempty"`
	Schemes       []string              `json:"schemes,omitempty"`
	BasePath      string                `json:"basePath,omitempty"`
	Tags          []Tag                 `json:"tags,omitempty"`
	Produces      []string              `json:"produces,omitempty"`
	Paths         map[string]*Path      `json:"paths,omitempty"`
	Definitions   map[string]*Schema    `json:"definitions,omitempty"`
	Parameters    map[string]*Parameter `json:"parameters,omitempty"`
	Extensions    []*Extension          `json:"x-extensions,omitempty"`
	GlobalOptions map[string]string     `json:"x-global-options,omitempty"`
}
type Info struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Extension struct {
	Base   string            `json:"base"`
	Fields []*ExtensionField `json:"fields"`
}

type ExtensionField struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Number int    `json:"number"`
}

type Path struct {
	Get        *Endpoint  `json:"get,omitempty"`
	Put        *Endpoint  `json:"put,omitempty"`
	Post       *Endpoint  `json:"post,omitempty"`
	Patch      *Endpoint  `json:"patch,omitempty"`
	Delete     *Endpoint  `json:"delete,omitempty"`
	Parameters Parameters `json:"parameters,omitempty"`
}

type Parameter struct {
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
	Format      string   `json:"format,omitempty"`
	In          string   `json:"in,omitempty"`
	Items       *Schema  `json:"items,omitempty"`
	Ref         string   `json:"$ref,omitempty"`
	Required    bool     `json:"required"`
	Schema      *Schema  `json:"schema,omitempty"`
	Type        string   `json:"type,omitempty"`
}

type Parameters []*Parameter

type Response struct {
	Description string  `json:"description"`
	Schema      *Schema `json:"schema,omitempty"`
}

type Endpoint struct {
	Summary     string               `yaml:"summary" json:"summary"`
	Description string               `yaml:"description" json:"description"`
	Parameters  Parameters           `yaml:"parameters" json:"parameters"`
	Tags        []string             `yaml:"tags" json:"tags,omitempty"`
	Responses   map[string]*Response `yaml:"responses" json:"responses"`
	OperationID string               `json:"operationId,omitempty"`
}

type Model struct {
	Properties map[string]*Schema `json:"properties"`
	Name       string
	Depth      int
}

type Schema struct {
	Description string `json:"description"`

	Ref    string   `json:"$ref,omitempty"`
	Type   string   `json:"type,omitempty"`
	Format string   `json:"format,omitempty"`
	Enum   []string `json:"enum,omitempty"`

	// objects
	Required   []string           `json:"required,omitempty"`
	Properties map[string]*Schema `json:"properties,omitempty"`

	// is an array
	Items *Schema `json:"items,omitempty"`

	Pattern   string `json:"pattern,omitempty"`
	MaxLength int    `json:"maxLength,omitempty"`
	MinLength int    `json:"minLength,omitempty"`
	Maximum   int    `json:"maximum,omitempty"`
	Minimum   int    `json:"minimum,omitempty"`
}
