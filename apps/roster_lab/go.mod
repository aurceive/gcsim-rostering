module github.com/genshinsim/gcsim/apps/roster_lab

go 1.25.0

require (
	github.com/genshinsim/gcsim/libs/enginebridge v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/genshinsim/gcsim/libs/enginebridge => ../../libs/enginebridge
