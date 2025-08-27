package models

type City struct {
	ID        int32  `json:"id"`
	Name      string `json:"name"`
	CountryID int32  `json:"country_id"`
}

type Country struct {
	ID         int32  `json:"id"`
	Name       string `json:"name"`
	Code       string `json:"code"`
	Alpha3Code string `json:"alpha3_code"`
	EuMember   bool   `json:"eu_member"`
	Continent  string `json:"continent"`
}
