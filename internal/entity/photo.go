package entity

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/photoprism/photoprism/internal/classify"
	"github.com/photoprism/photoprism/internal/event"
	"github.com/photoprism/photoprism/internal/form"
	"github.com/photoprism/photoprism/pkg/rnd"
	"github.com/photoprism/photoprism/pkg/txt"
	"github.com/ulule/deepcopier"
)

// Photo represents a photo, all its properties, and link to all its images and sidecar files.
type Photo struct {
	ID               uint         `gorm:"primary_key" yaml:"-"`
	TakenAt          time.Time    `gorm:"type:datetime;index:idx_photos_taken_uid;" json:"TakenAt" yaml:"TakenAt"`
	TakenAtLocal     time.Time    `gorm:"type:datetime;" yaml:"-"`
	TakenSrc         string       `gorm:"type:varbinary(8);" json:"TakenSrc" yaml:"TakenSrc,omitempty"`
	PhotoUID         string       `gorm:"type:varbinary(36);unique_index;index:idx_photos_taken_uid;" json:"UID" yaml:"UID"`
	PhotoType        string       `gorm:"type:varbinary(8);default:'image';" json:"Type" yaml:"Type"`
	PhotoTitle       string       `gorm:"type:varchar(255);" json:"Title" yaml:"Title"`
	TitleSrc         string       `gorm:"type:varbinary(8);" json:"TitleSrc" yaml:"TitleSrc,omitempty"`
	PhotoDescription string       `gorm:"type:text;" json:"Description" yaml:"Description,omitempty"`
	DescriptionSrc   string       `gorm:"type:varbinary(8);" json:"DescriptionSrc" yaml:"DescriptionSrc,omitempty"`
	Details          Details      `json:"Details" yaml:"Details"`
	PhotoPath        string       `gorm:"type:varbinary(768);index;" yaml:"-"`
	PhotoName        string       `gorm:"type:varbinary(255);" yaml:"-"`
	PhotoFavorite    bool         `json:"Favorite" yaml:"Favorite,omitempty"`
	PhotoPrivate     bool         `json:"Private" yaml:"Private,omitempty"`
	TimeZone         string       `gorm:"type:varbinary(64);" json:"TimeZone" yaml:"-"`
	PlaceUID         string       `gorm:"type:varbinary(16);index;" json:"PlaceUID" yaml:"-"`
	LocUID           string       `gorm:"type:varbinary(16);index;" json:"LocUID" yaml:"-"`
	LocSrc           string       `gorm:"type:varbinary(8);" json:"LocSrc" yaml:"-"`
	PhotoLat         float32      `gorm:"type:FLOAT;index;" json:"Lat" yaml:"Lat,omitempty"`
	PhotoLng         float32      `gorm:"type:FLOAT;index;" json:"Lng" yaml:"Lng,omitempty"`
	PhotoAltitude    int          `json:"Altitude" yaml:"Altitude,omitempty"`
	PhotoCountry     string       `gorm:"type:varbinary(2);index:idx_photos_country_year_month;default:'zz'" json:"Country" yaml:"-"`
	PhotoYear        int          `gorm:"index:idx_photos_country_year_month;" json:"Year" yaml:"-"`
	PhotoMonth       int          `gorm:"index:idx_photos_country_year_month;" json:"Month" yaml:"-"`
	PhotoIso         int          `json:"Iso" yaml:"ISO,omitempty"`
	PhotoExposure    string       `gorm:"type:varbinary(64);" json:"Exposure" yaml:"Exposure,omitempty"`
	PhotoFNumber     float32      `gorm:"type:FLOAT;" json:"FNumber" yaml:"FNumber,omitempty"`
	PhotoFocalLength int          `json:"FocalLength" yaml:"FocalLength,omitempty"`
	PhotoQuality     int          `gorm:"type:SMALLINT" json:"Quality" yaml:"-"`
	PhotoResolution  int          `gorm:"type:SMALLINT" json:"Resolution" yaml:"-"`
	CameraID         uint         `gorm:"index:idx_photos_camera_lens;" json:"CameraID" yaml:"-"`
	CameraSerial     string       `gorm:"type:varbinary(255);" json:"CameraSerial" yaml:"CameraSerial,omitempty"`
	CameraSrc        string       `gorm:"type:varbinary(8);" json:"CameraSrc" yaml:"-"`
	LensID           uint         `gorm:"index:idx_photos_camera_lens;" json:"LensID" yaml:"-"`
	Camera           *Camera      `gorm:"association_autoupdate:false;association_autocreate:false" json:"Camera" yaml:"-"`
	Lens             *Lens        `gorm:"association_autoupdate:false;association_autocreate:false" json:"Lens" yaml:"-"`
	Location         *Location    `gorm:"foreignkey:loc_uid;association_foreignkey:loc_uid;association_autoupdate:false;association_autocreate:false" json:"Location" yaml:"-"`
	Place            *Place       `gorm:"foreignkey:place_uid;association_foreignkey:place_uid;association_autoupdate:false;association_autocreate:false" json:"-" yaml:"-"`
	Links            []Link       `gorm:"foreignkey:share_uid;association_foreignkey:photo_uid" json:"Links" yaml:"-"`
	Keywords         []Keyword    `json:"-" yaml:"-"`
	Albums           []Album      `json:"-" yaml:"-"`
	Files            []File       `yaml:"-"`
	Labels           []PhotoLabel `yaml:"-"`
	CreatedAt        time.Time    `yaml:"CreatedAt,omitempty"`
	UpdatedAt        time.Time    `yaml:"UpdatedAt,omitempty"`
	EditedAt         *time.Time   `yaml:"EditedAt,omitempty"`
	DeletedAt        *time.Time   `sql:"index" yaml:"DeletedAt,omitempty"`
}

// SavePhotoForm saves a model in the database using form data.
func SavePhotoForm(model Photo, form form.Photo, geoApi string) error {
	db := Db()
	locChanged := model.PhotoLat != form.PhotoLat || model.PhotoLng != form.PhotoLng

	if err := deepcopier.Copy(&model).From(form); err != nil {
		return err
	}

	if !model.HasID() {
		return errors.New("photo: can't save form, id is empty")
	}

	model.UpdateYearMonth()

	if form.Details.PhotoID == model.ID {
		if err := deepcopier.Copy(&model.Details).From(form.Details); err != nil {
			return err
		}

		model.Details.Keywords = strings.Join(txt.UniqueKeywords(model.Details.Keywords), ", ")
	}

	if model.HasLatLng() && locChanged && model.LocSrc == SrcManual {
		locKeywords, labels := model.UpdateLocation(geoApi)

		model.AddLabels(labels)

		w := txt.UniqueKeywords(model.Details.Keywords)
		w = append(w, locKeywords...)

		model.Details.Keywords = strings.Join(txt.UniqueWords(w), ", ")
	}

	if err := model.UpdateTitle(model.ClassifyLabels()); err != nil {
		log.Warnf("%s (%s)", err.Error(), model.PhotoUID)
	}

	if err := model.IndexKeywords(); err != nil {
		log.Error(err)
	}

	edited := time.Now().UTC()
	model.EditedAt = &edited
	model.PhotoQuality = model.QualityScore()

	if err := db.Unscoped().Save(&model).Error; err != nil {
		return err
	}

	if err := UpdatePhotoCounts(); err != nil {
		log.Errorf("photo: %s", err)
	}

	return nil
}

// Save the entity in the database.
func (m *Photo) Save() error {
	if !m.HasID() {
		return errors.New("photo: can't save to database, id is empty")
	}

	db := Db()
	labels := m.ClassifyLabels()

	m.UpdateYearMonth()

	if err := m.UpdateTitle(labels); err != nil {
		log.Warnf("%s (%s)", err.Error(), m.PhotoUID)
	}

	if m.DetailsLoaded() {
		w := txt.UniqueKeywords(m.Details.Keywords)
		w = append(w, labels.Keywords()...)
		m.Details.Keywords = strings.Join(txt.UniqueWords(w), ", ")
	}

	if err := m.IndexKeywords(); err != nil {
		log.Error(err)
	}

	m.PhotoQuality = m.QualityScore()

	if err := db.Unscoped().Save(m).Error; err != nil {
		return err
	}

	if err := UpdatePhotoCounts(); err != nil {
		log.Errorf("photo: %s", err)
	}

	return nil
}

// ClassifyLabels returns all associated labels as classify.Labels
func (m *Photo) ClassifyLabels() classify.Labels {
	result := classify.Labels{}

	for _, l := range m.Labels {
		if l.Label == nil {
			log.Warnf("photo: empty reference while creating classify labels (%d -> %d)", l.PhotoID, l.LabelID)
			continue
		}

		result = append(result, l.ClassifyLabel())
	}

	return result
}

// BeforeCreate creates a random UID if needed before inserting a new row to the database.
func (m *Photo) BeforeCreate(scope *gorm.Scope) error {
	if m.TakenAt.IsZero() || m.TakenAtLocal.IsZero() {
		now := time.Now()

		if err := scope.SetColumn("TakenAt", now); err != nil {
			return err
		}

		if err := scope.SetColumn("TakenAtLocal", now); err != nil {
			return err
		}
	}

	if rnd.IsPPID(m.PhotoUID, 'p') {
		return nil
	}

	return scope.SetColumn("PhotoUID", rnd.PPID('p'))
}

// BeforeSave ensures the existence of TakenAt properties before indexing or updating a photo
func (m *Photo) BeforeSave(scope *gorm.Scope) error {
	if m.TakenAt.IsZero() || m.TakenAtLocal.IsZero() {
		now := time.Now()

		if err := scope.SetColumn("TakenAt", now); err != nil {
			return err
		}

		if err := scope.SetColumn("TakenAtLocal", now); err != nil {
			return err
		}
	}

	return nil
}

// IndexKeywords adds given keywords to the photo entry
func (m *Photo) IndexKeywords() error {
	if !m.DetailsLoaded() {
		return fmt.Errorf("photo: can't index keywords, details not loaded (%s)", m.PhotoUID)
	}

	db := Db()

	var keywordIds []uint
	var keywords []string

	// Add title, description and other keywords
	keywords = append(keywords, txt.Keywords(m.PhotoTitle)...)
	keywords = append(keywords, txt.Keywords(m.PhotoDescription)...)
	keywords = append(keywords, txt.Keywords(m.Details.Keywords)...)
	keywords = append(keywords, txt.Keywords(m.Details.Subject)...)
	keywords = append(keywords, txt.Keywords(m.Details.Artist)...)

	keywords = txt.UniqueWords(keywords)

	for _, w := range keywords {
		kw := NewKeyword(w).FirstOrCreate()

		if kw.Skip {
			continue
		}

		keywordIds = append(keywordIds, kw.ID)

		NewPhotoKeyword(m.ID, kw.ID).FirstOrCreate()
	}

	return db.Where("photo_id = ? AND keyword_id NOT IN (?)", m.ID, keywordIds).Delete(&PhotoKeyword{}).Error
}

// PreloadFiles prepares gorm scope to retrieve photo file
func (m *Photo) PreloadFiles() {
	q := Db().NewScope(nil).DB().
		Table("files").
		Select(`files.*`).
		Where("files.photo_id = ?", m.ID).
		Order("files.file_name DESC")

	logError(q.Scan(&m.Files))
}

/* func (m *Photo) PreloadLabels() {
	q := Db().NewScope(nil).DB().
		Table("labels").
		Select(`labels.*`).
		Joins("JOIN photos_labels ON photos_labels.label_id = labels.id AND photos_labels.photo_id = ?", m.ID).
		Where("labels.deleted_at IS NULL").
		Order("labels.label_name ASC")

	logError(q.Scan(&m.Labels))
} */

// PreloadKeywords prepares gorm scope to retrieve photo keywords
func (m *Photo) PreloadKeywords() {
	q := Db().NewScope(nil).DB().
		Table("keywords").
		Select(`keywords.*`).
		Joins("JOIN photos_keywords ON photos_keywords.keyword_id = keywords.id AND photos_keywords.photo_id = ?", m.ID).
		Order("keywords.keyword ASC")

	logError(q.Scan(&m.Keywords))
}

// PreloadAlbums prepares gorm scope to retrieve photo albums
func (m *Photo) PreloadAlbums() {
	q := Db().NewScope(nil).DB().
		Table("albums").
		Select(`albums.*`).
		Joins("JOIN photos_albums ON photos_albums.album_uid = albums.album_uid AND photos_albums.photo_uid = ?", m.PhotoUID).
		Where("albums.deleted_at IS NULL").
		Order("albums.album_title ASC")

	logError(q.Scan(&m.Albums))
}

// PreloadMany prepares gorm scope to retrieve photo file, albums and keywords
func (m *Photo) PreloadMany() {
	m.PreloadFiles()
	// m.PreloadLabels()
	m.PreloadKeywords()
	m.PreloadAlbums()
}

// HasID checks if the photo has a database id and uid.
func (m *Photo) HasID() bool {
	return m.ID > 0 && m.PhotoUID != ""
}

// NoLocation checks if the photo has an unknown location.
func (m *Photo) NoLocation() bool {
	return m.LocUID == "" || m.LocUID == UnknownLocation.LocUID
}

// HasLocation checks if the photo has a known location.
func (m *Photo) HasLocation() bool {
	return !m.NoLocation()
}

// LocationLoaded checks if the photo has a known location that is currently loaded.
func (m *Photo) LocationLoaded() bool {
	return m.Location != nil && m.Location.Place != nil && m.Location.LocUID != UnknownLocation.LocUID
}

// HasLatLng checks if the photo has a latitude and longitude.
func (m *Photo) HasLatLng() bool {
	return m.PhotoLat != 0 && m.PhotoLng != 0
}

// NoLatLng checks if latitude and longitude are missing.
func (m *Photo) NoLatLng() bool {
	return !m.HasLatLng()
}

// NoPlace checks if the photo has an unknown place.
func (m *Photo) NoPlace() bool {
	return m.PlaceUID == "" || m.PlaceUID == UnknownPlace.PlaceUID
}

// HasPlace checks if the photo has a known place.
func (m *Photo) HasPlace() bool {
	return !m.NoPlace()
}

// NoTitle checks if the photo has no Title
func (m *Photo) NoTitle() bool {
	return m.PhotoTitle == ""
}

// NoCameraSerial checks if the photo has no CameraSerial
func (m *Photo) NoCameraSerial() bool {
	return m.CameraSerial == ""
}

// HasTitle checks if the photo has a title.
func (m *Photo) HasTitle() bool {
	return m.PhotoTitle != ""
}

// HasDescription checks if the photo has a description.
func (m *Photo) HasDescription() bool {
	return m.PhotoDescription != ""
}

// DetailsLoaded returns true if photo details exist.
func (m *Photo) DetailsLoaded() bool {
	return m.Details.PhotoID == m.ID
}

// UpdateTitle updated the photo title based on location and labels.
func (m *Photo) UpdateTitle(labels classify.Labels) error {
	if m.TitleSrc != SrcAuto && m.HasTitle() {
		return errors.New("photo: won't update title, was modified")
	}

	knownLocation := m.LocationLoaded()

	if knownLocation {
		loc := m.Location

		if title := labels.Title(loc.Name()); title != "" { // TODO: User defined title format
			log.Infof("photo: using label %s to create photo title", txt.Quote(title))
			if loc.NoCity() || loc.LongCity() || loc.CityContains(title) {
				m.SetTitle(fmt.Sprintf("%s / %s / %s", txt.Title(title), loc.CountryName(), m.TakenAt.Format("2006")), SrcAuto)
			} else {
				m.SetTitle(fmt.Sprintf("%s / %s / %s", txt.Title(title), loc.City(), m.TakenAt.Format("2006")), SrcAuto)
			}
		} else if loc.Name() != "" && loc.City() != "" {
			if len(loc.Name()) > 45 {
				m.SetTitle(txt.Title(loc.Name()), SrcAuto)
			} else if len(loc.Name()) > 20 || len(loc.City()) > 16 || strings.Contains(loc.Name(), loc.City()) {
				m.SetTitle(fmt.Sprintf("%s / %s", loc.Name(), m.TakenAt.Format("2006")), SrcAuto)
			} else {
				m.SetTitle(fmt.Sprintf("%s / %s / %s", loc.Name(), loc.City(), m.TakenAt.Format("2006")), SrcAuto)
			}
		} else if loc.City() != "" && loc.CountryName() != "" {
			if len(loc.City()) > 20 {
				m.SetTitle(fmt.Sprintf("%s / %s", loc.City(), m.TakenAt.Format("2006")), SrcAuto)
			} else {
				m.SetTitle(fmt.Sprintf("%s / %s / %s", loc.City(), loc.CountryName(), m.TakenAt.Format("2006")), SrcAuto)
			}
		}
	}

	if !knownLocation || m.NoTitle() {
		if len(labels) > 0 && labels[0].Priority >= -1 && labels[0].Uncertainty <= 85 && labels[0].Name != "" {
			if m.TakenSrc != SrcAuto {
				m.SetTitle(fmt.Sprintf("%s / %s", txt.Title(labels[0].Name), m.TakenAt.Format("2006")), SrcAuto)
			} else {
				m.SetTitle(txt.Title(labels[0].Name), SrcAuto)
			}
		} else if !m.TakenAtLocal.IsZero() && m.TakenSrc != SrcAuto {
			m.SetTitle(fmt.Sprintf("%s / %s", TitleUnknown, m.TakenAtLocal.Format("2006")), SrcAuto)
		} else {
			m.SetTitle(TitleUnknown, SrcAuto)
		}

		log.Infof("photo: changed photo title to %s", txt.Quote(m.PhotoTitle))
	} else {
		log.Infof("photo: new title is %s", txt.Quote(m.PhotoTitle))
	}

	return nil
}

// AddLabels updates the entity with additional or updated label information.
func (m *Photo) AddLabels(labels classify.Labels) {
	// TODO: Update classify labels from database
	for _, label := range labels {
		lm := NewLabel(label.Title(), label.Priority).FirstOrCreate()

		if lm.New {
			event.EntitiesCreated("labels", []*Label{lm})

			if label.Priority >= 0 {
				event.Publish("count.labels", event.Data{
					"count": 1,
				})
			}
		}

		if err := lm.Update(label); err != nil {
			log.Errorf("index: %s", err)
		}

		plm := NewPhotoLabel(m.ID, lm.ID, label.Uncertainty, label.Source).FirstOrCreate()

		if plm.Uncertainty > label.Uncertainty && plm.Uncertainty < 100 {
			plm.Uncertainty = label.Uncertainty
			plm.LabelSrc = label.Source
			if err := Db().Save(&plm).Error; err != nil {
				log.Errorf("index: %s", err)
			}
		}
	}

	Db().Set("gorm:auto_preload", true).Model(m).Related(&m.Labels)
}

// SetTitle changes the photo title and clips it to 300 characters.
func (m *Photo) SetTitle(title, source string) {
	newTitle := txt.Clip(title, txt.ClipDefault)

	if newTitle == "" {
		return
	}

	if m.TitleSrc != SrcAuto && m.TitleSrc != source && source != SrcManual && m.HasTitle() {
		return
	}

	m.PhotoTitle = newTitle
	m.TitleSrc = source
}

// SetDescription changes the photo description if not empty and from the same source.
func (m *Photo) SetDescription(desc, source string) {
	newDesc := txt.Clip(desc, txt.ClipDescription)

	if newDesc == "" {
		return
	}

	if m.DescriptionSrc != SrcAuto && m.DescriptionSrc != source && source != SrcManual && m.PhotoDescription != "" {
		return
	}

	m.PhotoDescription = newDesc
	m.DescriptionSrc = source
}

// SetTakenAt changes the photo date if not empty and from the same source.
func (m *Photo) SetTakenAt(taken, local time.Time, zone, source string) {
	if taken.IsZero() || taken.Year() < 1000 {
		return
	}

	if m.TakenSrc != SrcAuto && m.TakenSrc != source && source != SrcManual {
		return
	}

	m.TakenAt = taken.Round(time.Second).UTC()
	m.TakenSrc = source

	if local.IsZero() || local.Year() < 1000 {
		m.TakenAtLocal = m.TakenAt
	} else {
		m.TakenAtLocal = local.Round(time.Second)
	}

	if zone != "" {
		m.TimeZone = zone
	}

	m.UpdateYearMonth()
}

// UpdateYearMonth updates internal date fields.
func (m *Photo) UpdateYearMonth() {
	if m.TakenAt.IsZero() || m.TakenAt.Year() < 1000 {
		return
	}

	if m.TakenAtLocal.IsZero() || m.TakenAtLocal.Year() < 1000 {
		m.TakenAtLocal = m.TakenAt
	}

	if m.TakenSrc == SrcAuto {
		m.PhotoYear = YearUnknown
		m.PhotoMonth = MonthUnknown
	} else {
		m.PhotoYear = m.TakenAtLocal.Year()
		m.PhotoMonth = int(m.TakenAtLocal.Month())
	}
}

// SetCoordinates changes the photo lat, lng and altitude if not empty and from the same source.
func (m *Photo) SetCoordinates(lat, lng float32, altitude int, source string) {
	if lat == 0 && lng == 0 {
		return
	}

	if m.LocSrc != SrcAuto && m.LocSrc != source && source != SrcManual {
		return
	}

	m.PhotoLat = lat
	m.PhotoLng = lng
	m.PhotoAltitude = altitude
	m.LocSrc = source
}

// AllFilesMissing returns true, if all files for this photo are missing.
func (m *Photo) AllFilesMissing() bool {
	count := 0

	if err := Db().Model(&File{}).
		Where("photo_id = ? AND b.file_missing = 0", m.ID).
		Count(&count).Error; err != nil {
		log.Error(err)
	}

	return count == 0
}

// Delete deletes the entity from the database.
func (m *Photo) Delete(permanently bool) error {
	if permanently {
		return m.DeletePermanently()
	}

	Db().Delete(File{}, "photo_id = ?", m.ID)

	return Db().Delete(m).Error
}

// Delete permanently deletes the entity from the database.
func (m *Photo) DeletePermanently() error {
	Db().Unscoped().Delete(File{}, "photo_id = ?", m.ID)
	Db().Unscoped().Delete(PhotoKeyword{}, "photo_id = ?", m.ID)
	Db().Unscoped().Delete(PhotoLabel{}, "photo_id = ?", m.ID)
	Db().Unscoped().Delete(PhotoAlbum{}, "photo_uid = ?", m.PhotoUID)

	return Db().Unscoped().Delete(m).Error
}

// NoDescription returns true if the photo has no description.
func (m *Photo) NoDescription() bool {
	return m.PhotoDescription == ""
}

// Updates a model attribute.
func (m *Photo) Update(attr string, value interface{}) error {
	return UnscopedDb().Model(m).UpdateColumn(attr, value).Error
}
