package utils

type IPType int
type IPSource string

type Subdivision struct {
	GeoNameID uint              `json:"geoname_id"`
	IsoCode   string            `json:"iso_code"`
	Names     map[string]string `json:"names"`
}

type City struct {
	GeoNameID uint              `json:"geoname_id"`
	Names     map[string]string `json:"names"`
}

type Continent struct {
	Code      string            `json:"code"`
	GeoNameID uint              `json:"geoname_id"`
	Names     map[string]string `json:"names"`
}

type Country struct {
	GeoNameID         uint              `json:"geoname_id"`
	IsInEuropeanUnion bool              `json:"is_in_european_union"`
	IsoCode           string            `json:"iso_code"`
	Names             map[string]string `json:"names"`
	Type              string            `json:"type"`
}

type Location struct {
	AccuracyRadius uint16  `json:"accuracy_radius"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	MetroCode      uint    `json:"metro_code"`
	TimeZone       string  `json:"time_zone"`
}

type Traits struct {
	IsAnonymousProxy    bool `json:"is_anonymous_proxy"`
	IsSatelliteProvider bool `json:"is_satellite_provider"`
}

type Postal struct {
	Code string `json:"code"`
}

type ASN struct {
	AutonomousSystemNumber       uint   `json:"autonomous_system_number"`
	AutonomousSystemOrganization string `json:"autonomous_system_organization"`
}

type AnonymousIP struct {
	IsAnonymous       bool `json:"is_anonymous"`
	IsAnonymousVPN    bool `json:"is_anonymous_vpn"`
	IsHostingProvider bool `json:"is_hosting_provider"`
	IsPublicProxy     bool `json:"is_public_proxy"`
	IsTorExitNode     bool `json:"is_tor_exit_node"`
}

type IPInfo struct {
	Address            string         `json:"address"`
	Source             string         `json:"source"`
	IsFallback         bool           `json:"is_fallback"`
	HasCity            bool           `json:"has_city"`
	City               *City          `json:"city"`
	Continent          *Continent     `json:"continent"`
	Country            *Country       `json:"country"`
	Location           *Location      `json:"location"`
	Postal             *Postal        `json:"postal"`
	RegisteredCountry  *Country       `json:"registered_country"`
	RepresentedCountry *Country       `json:"represented_country"`
	Subdivisions       []*Subdivision `json:"subdivisions"`
	Traits             *Traits        `json:"traits"`
	HasASN             bool           `json:"has_asn"`
	ASN                *ASN           `json:"asn"`
	HasAnonymousIP     bool           `json:"has_anonymous_ip"`
	AnonymousIP        *AnonymousIP   `json:"anonymous_ip"`
}
