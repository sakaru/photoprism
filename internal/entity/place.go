package entity

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/photoprism/photoprism/internal/maps"
	"github.com/photoprism/photoprism/pkg/txt"
)

// Place used to associate photos to places
type Place struct {
	PlaceUID    string    `gorm:"type:varbinary(16);primary_key;auto_increment:false;" json:"PlaceUID" yaml:"PlaceUID"`
	LocLabel    string    `gorm:"type:varbinary(768);unique_index;" json:"Label" yaml:"Label"`
	LocCity     string    `gorm:"type:varchar(255);" json:"City" yaml:"City,omitempty"`
	LocState    string    `gorm:"type:varchar(255);" json:"State" yaml:"State,omitempty"`
	LocCountry  string    `gorm:"type:varbinary(2);" json:"Country" yaml:"Country,omitempty"`
	LocKeywords string    `gorm:"type:varchar(255);" json:"Keywords" yaml:"Keywords,omitempty"`
	LocNotes    string    `gorm:"type:text;" json:"Notes" yaml:"Notes,omitempty"`
	LocFavorite bool      `json:"Favorite" yaml:"Favorite,omitempty"`
	PhotoCount  int       `json:"PhotoCount" yaml:"-"`
	CreatedAt   time.Time `json:"CreatedAt" yaml:"-"`
	UpdatedAt   time.Time `json:"UpdatedAt" yaml:"-"`
	New         bool      `gorm:"-" json:"-" yaml:"-"`
}

// UnknownPlace is PhotoPrism's default place.
var UnknownPlace = Place{
	PlaceUID:    "zz",
	LocLabel:    "Unknown",
	LocCity:     "Unknown",
	LocState:    "Unknown",
	LocCountry:  "zz",
	LocKeywords: "",
	LocNotes:    "",
	LocFavorite: false,
	PhotoCount:  -1,
}

// CreateUnknownPlace creates the default place if not exists.
func CreateUnknownPlace() {
	FirstOrCreatePlace(&UnknownPlace)
}

// AfterCreate sets the New column used for database callback
func (m *Place) AfterCreate(scope *gorm.Scope) error {
	m.New = true
	return nil
}

// FindPlaceByLabel returns a place from an id or a label
func FindPlaceByLabel(uid string, label string) *Place {
	place := &Place{}

	if label == "" {
		if err := Db().First(place, "place_uid = ?", uid).Error; err != nil {
			log.Debugf("place: %s for uid %s", err.Error(), uid)
			return nil
		}
	} else if err := Db().First(place, "place_uid = ? OR loc_label = ?", uid, label).Error; err != nil {
		log.Debugf("place: %s for uid %s / label %s", err.Error(), uid, txt.Quote(label))
		return nil
	}

	return place
}

// Find returns db record of place
func (m *Place) Find() error {
	if err := Db().First(m, "place_uid = ?", m.PlaceUID).Error; err != nil {
		return err
	}

	return nil
}

// Create inserts a new row to the database.
func (m *Place) Create() error {
	if err := Db().Create(m).Error; err != nil {
		return err
	}

	return nil
}

// FirstOrCreatePlace inserts a new row if not exists.
func FirstOrCreatePlace(m *Place) *Place {
	if m.PlaceUID == "" {
		log.Errorf("place: uid must not be empty")
		return nil
	}

	if m.LocLabel == "" {
		log.Errorf("place: label must not be empty (uid %s)", m.PlaceUID)
		return nil
	}

	result := Place{}

	if err := Db().Where("place_uid = ? OR loc_label = ?", m.PlaceUID, m.LocLabel).First(&result).Error; err == nil {
		return &result
	} else if err := m.Create(); err != nil {
		log.Errorf("place: %s", err)
		return nil
	}

	return m
}

// Unknown returns true if this is an unknown place
func (m Place) Unknown() bool {
	return m.PlaceUID == UnknownPlace.PlaceUID
}

// Label returns place label
func (m Place) Label() string {
	return m.LocLabel
}

// City returns place City
func (m Place) City() string {
	return m.LocCity
}

// State returns place State
func (m Place) State() string {
	return m.LocState
}

// CountryCode returns place CountryCode
func (m Place) CountryCode() string {
	return m.LocCountry
}

// CountryName returns place CountryName
func (m Place) CountryName() string {
	return maps.CountryNames[m.LocCountry]
}

// Notes returns place Notes
func (m Place) Notes() string {
	return m.LocNotes
}
