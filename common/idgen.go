package common

import "github.com/yitter/idgenerator-go/idgen"

func InitIdGenerator() {
	options := idgen.NewIdGeneratorOptions(1)
	idgen.SetIdGenerator(options)
}

func GetNextId() int {
	return int(idgen.NextId())
}