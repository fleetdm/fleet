// Copyright (c) Facebook, Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package schema

// ReportIndexItem one item in array returned on /report/index
// we only want the reportID so we can use it to get /report/{reportID}
type ReportIndexItem struct {
	ReportID string `json:"reportId"`
}

// ReportWrapper struct
type ReportWrapper struct {
	Report Report `json:"report"`
}

// Report struct
type Report struct {
	Audience            []string           `json:"audience"`
	Copyright           string             `json:"copyright"`
	CveIDs              *ReportCveIDs      `json:"cveIds"`
	ExecSummary         string             `json:"execSummary"`
	IntelligenceType    string             `json:"intelligenceType"`
	PublishDate         string             `json:"publishDate"`
	ReportID            string             `json:"reportId"`
	ReportType          string             `json:"reportType"`
	TagSection          *ReportTagSection  `json:"tagSection"`
	ThreatScape         *ReportThreatScape `json:"ThreatScape"`
	Title               string             `json:"title"`
	Version             string             `json:"version"`
	Version1PublishDate string             `json:"version1PublishDate"`
}

// ReportCveIDs struct
type ReportCveIDs struct {
	CveID []string `json:"cveId"`
}

// ReportTagSection struct
type ReportTagSection struct {
	Files    *ReportFiles    `json:"files"`
	Main     *ReportMain     `json:"main"`
	Networks *ReportNetworks `json:"networks"`
}

// ReportThreatScape struct
type ReportThreatScape struct {
	Product []string `json:"product"`
}

// ReportFiles struct
type ReportFiles struct {
	File []*ReportFile `json:"file"`
}

// ReportFile struct
type ReportFile struct {
	Sha1       string `json:"sha1"`
	Identifier string `json:"identifier"`
	Actor      string `json:"actor"`
	FileName   string `json:"fileName"`
	FileSize   string `json:"fileSize"`
	ActorID    string `json:"actorId"`
	Sha256     string `json:"sha256"`
	Type       string `json:"type"`
	Md5        string `json:"md5"`
}

// ReportMain struct
type ReportMain struct {
	Actors               *ReportActors               `json:"actors"`
	AffectedIndustries   *ReportAffectedIndustries   `json:"affectedIndustries"`
	IntendedEffects      *ReportIntendedEffects      `json:"intendedEffects"`
	MalwareFamilies      *ReportMalwareFamilies      `json:"malwareFamilies"`
	Motivations          *ReportMotivations          `json:"motivations"`
	SourceGeographies    *ReportSourceGeographies    `json:"sourceGeographies"`
	TargetedInformations *ReportTargetedInformations `json:"targetedInformations"`
	TargetGeographies    *ReportTargetGeographies    `json:"targetGeographies"`
	Ttps                 *ReportTtps                 `json:"ttps"`
}

// ReportActors struct
type ReportActors struct {
	Actor []*ReportActor `json:"actor"`
}

// ReportActor struct
type ReportActor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ReportNetworks struct
type ReportNetworks struct {
	Networks []*ReportNetwork `json:"network"`
}

// ReportNetwork struct
type ReportNetwork struct {
	URL         string `json:"url"`
	NetworkType string `json:"networkType"`
	Identifier  string `json:"identifier"`
	Actor       string `json:"actor"`
	ActorID     string `json:"actorId"`
	Domain      string `json:"domain"`
}

// ReportMotivations struct
type ReportMotivations struct {
	Motivation []string `json:"motivation"`
}

// ReportSourceGeographies struct
type ReportSourceGeographies struct {
	SourceGeography []string `json:"sourceGeography"`
}

// ReportAffectedIndustries struct
type ReportAffectedIndustries struct {
	AffectedIndustry []string `json:"affectedIndustry"`
}

// ReportIntendedEffects struct
type ReportIntendedEffects struct {
	IntendedEffect []string `json:"intendedEffect"`
}

// ReportTtps struct
type ReportTtps struct {
	Ttp []string `json:"ttp"`
}

// ReportTargetGeographies struct
type ReportTargetGeographies struct {
	TargetGeography []string `json:"targetGeography"`
}

// ReportTargetedInformations struct
type ReportTargetedInformations struct {
	TargetedInformation []string `json:"targetedInformation"`
}

// ReportMalwareFamilies struct
type ReportMalwareFamilies struct {
	MalwareFamily []*ReportMalwareFamily `json:"malwareFamily"`
}

// ReportMalwareFamily struct
type ReportMalwareFamily struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
