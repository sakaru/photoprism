package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFolder(t *testing.T) {
	t.Run("2020/05", func(t *testing.T) {
		folder := NewFolder(RootOriginals, "2020/05", nil)
		assert.Equal(t, RootOriginals, folder.Root)
		assert.Equal(t, "2020/05", folder.Path)
		assert.Equal(t, "May 2020", folder.FolderTitle)
		assert.Equal(t, "", folder.FolderDescription)
		assert.Equal(t, "", folder.FolderType)
		assert.Equal(t, SortOrderName, folder.FolderOrder)
		assert.IsType(t, "", folder.FolderUID)
		assert.Equal(t, false, folder.FolderFavorite)
		assert.Equal(t, false, folder.FolderIgnore)
		assert.Equal(t, false, folder.FolderWatch)
	})

	t.Run("/2020/05/01/", func(t *testing.T) {
		folder := NewFolder(RootOriginals, "/2020/05/01/", nil)
		assert.Equal(t, "2020/05/01", folder.Path)
		assert.Equal(t, "May 2020", folder.FolderTitle)
	})

	t.Run("/2020/05/23/", func(t *testing.T) {
		folder := NewFolder(RootImport, "/2020/05/23/", nil)
		assert.Equal(t, "2020/05/23", folder.Path)
		assert.Equal(t, "May 23, 2020", folder.FolderTitle)
	})

	t.Run("/2020/05/23 Birthday", func(t *testing.T) {
		folder := NewFolder(RootOriginals, "/2020/05/23 Birthday", nil)
		assert.Equal(t, "2020/05/23 Birthday", folder.Path)
		assert.Equal(t, "23 Birthday", folder.FolderTitle)
	})

	t.Run("empty", func(t *testing.T) {
		folder := NewFolder(RootOriginals, "", nil)
		assert.Equal(t, "", folder.Path)
		assert.Equal(t, "Originals", folder.FolderTitle)
	})

	t.Run("root", func(t *testing.T) {
		folder := NewFolder(RootOriginals, RootPath, nil)
		assert.Equal(t, "", folder.Path)
		assert.Equal(t, "Originals", folder.FolderTitle)
	})
}
