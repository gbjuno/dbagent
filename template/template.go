package template

const (
	DOCKER_Single = `net:
  port: {{ .Spec.Port }}
operationProfiling: {}
processManagement:
  fork: false
  pidFilePath: /data/mongodb-{{ .GetName }}.pid
storage:
  dbPath: /data/mongodb-{{ .GetName }}
  engine: wiredTiger
  journal:
     enabled: true
systemLog:
  destination: file
  path: /data/mongodb-{{ .GetName }}.log
`

	DOCKER_Replset = `net:
  port: {{ .Spec.Port }}
operationProfiling: {}
processManagement:
  fork: false
  pidFilePath: /data/mongodb-{{ .GetName }}.pid
replication:
  replSetName: {{ .GetName }}
storage:
  dbPath: /data/mongodb-{{ .GetName }}
  engine: wiredTiger
  journal:
     enabled: true
systemLog:
  destination: file
  path: /data/mongodb-{{ .GetName }}.log
`

	NATIVE_Single = `net:
  port: {{ .Spec.Port }}
operationProfiling: {}
processManagement:
  fork: true
  pidFilePath: {{ .Status.DataPath }}/mongodb-{{ .GetName }}.pid
storage:
  dbPath: {{ .Status.DataPath }}/mongodb-{{ .GetName }}
  engine: wiredTiger
  journal:
     enabled: true
systemLog:
  destination: file
  path: {{ .Status.DataPath }}/mongodb-{{ .GetName }}.log
`

	NATIVE_Replset = `net:
  port: {{ .Spec.Port }}
operationProfiling: {}
processManagement:
  fork: true
  pidFilePath: {{ .Status.DataPath }}/mongodb-{{ .GetName }}.pid
replication:
  replSetName: {{ .GetName }}
storage:
  dbPath: {{ .Status.DataPath }}/mongodb-{{ .GetName }}
  engine: wiredTiger
  journal:
     enabled: true
systemLog:
  destination: file
  path: {{ .Status.DataPath }}/mongodb-{{ .GetName }}.log
`
)
