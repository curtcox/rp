package model

var ConfidenceRank = map[string]int{
	"unsupported":              0,
	"claimed":                  1,
	"observed":                 2,
	"attested":                 3,
	"reproduced":               4,
	"independently_reproduced": 5,
}

func ConfidenceAtLeast(got, min string) bool {
	return ConfidenceRank[got] >= ConfidenceRank[min]
}

func KnownConfidence(confidence string) bool {
	_, ok := ConfidenceRank[confidence]
	return ok
}
