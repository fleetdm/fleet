package service

import (
	"testing"

	"github.com/beevik/etree"
	"github.com/stretchr/testify/assert"
)

func TestPrepareWindowsMDMCommand(t *testing.T) {
	c := &Client{}

	validXML := []byte(`
	<Exec>
		<CmdID>some-id</CmdID>
		<Item></Item>
	</Exec>
	`)

	invalidCmdXML := []byte(`
	<Add>
		<CmdID>some-id</Cmd>
		<Item></Item>
	</Add>
	`)

	noCmdIDXML := []byte(`
	<Exec>
		<Item></Item>
	</Exec>
	`)

	t.Run("Modifies valid CmdID", func(t *testing.T) {
		modified, err := c.prepareWindowsMDMCommand(validXML)
		assert.Nil(t, err)

		doc := etree.NewDocument()
		err = doc.ReadFromBytes(modified)
		assert.Nil(t, err)

		element := doc.FindElement("//CmdID")
		assert.NotNil(t, element)
		assert.NotEmpty(t, element.Text())
	})

	t.Run("Adds CmdID if missing", func(t *testing.T) {
		modified, err := c.prepareWindowsMDMCommand(noCmdIDXML)
		assert.Nil(t, err)

		doc := etree.NewDocument()
		err = doc.ReadFromBytes(modified)
		assert.Nil(t, err)

		element := doc.FindElement("//CmdID")
		assert.NotNil(t, element)
		assert.NotEmpty(t, element.Text())
	})

	t.Run("Returns error on invalid XML", func(t *testing.T) {
		_, err := c.prepareWindowsMDMCommand(invalidCmdXML)
		assert.NotNil(t, err)

		_, err = c.prepareWindowsMDMCommand([]byte("<Exec><Exec"))
		assert.NotNil(t, err)
	})
}
