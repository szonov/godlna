package device

import (
	"encoding/xml"
	"strings"
)

const (
	DeviceXMLNamespace     = "urn:schemas-upnp-org:device-1-0"
	DlnaDeviceXMLNamespace = "urn:schemas-dlna-org:device-1-0"
	DlnaSecXMLNamespace    = "http://www.sec.co.kr/dlna"
)

// reference https://upnp.org/specs/arch/UPnP-arch-DeviceArchitecture-v1.0-20080424.pdf
// on the page 26 there is XML listing and then detailed description

type SpecVersion struct {
	// Required. Major version of the UPnP Device Architecture. Must be 1.
	Major uint `xml:"major"`

	// Required. Minor version of the UPnP Device Architecture.
	// Must be 0 in devices that implement UDA version 1.0.
	// Must accurately reflect the version number of the UPnP Device Architecture supported by
	// the device. Control points must be prepared to accept a higher version number than
	// the control point itself implements.
	Minor uint `xml:"minor"`
}

type Icon struct {
	// Mimetype Required. Icon's MIME type
	Mimetype string `xml:"mimetype"`

	// Width Required. Horizontal dimension of icon in pixels. Integer.
	Width int `xml:"width"`

	// Height Required. Vertical dimension of icon in pixels. Integer.
	Height int `xml:"height"`

	// Depth Required. Number of color bits per pixel. Integer.
	Depth int `xml:"depth"`

	// URL Required. Pointer to icon image.
	// (XML does not support direct embedding of binary data) Retrieved via HTTP.
	// May be relative to base URL. Single URL.
	URL string `xml:"url"`
}

type Service struct {
	XMLName xml.Name `xml:"service"`
	// ServiceType Required. UPnP service type.
	// Must not contain a hash character (#, 23 Hex in UTF-8).
	ServiceType string `xml:"serviceType"`

	// ServiceId Required. Service identifier.
	// Must be unique within this device description.
	ServiceId string `xml:"serviceId"`

	// SCPDURL Required. URL for service description
	// (Service Control Protocol Definition URL). Single URL.
	SCPDURL string

	// ControlURL Required. URL for control. Single URL.
	ControlURL string `xml:"controlURL"`

	// EventSubURL Required. URL for eventing. Must be unique
	// within the device; no two services may have the same URL for eventing.
	// If the service has no evented variables, it should not have eventing;
	// if the service does not have eventing, this element must be present but should be empty,
	// i.e.,<eventSubURL></eventSubURL>. Single URL.
	EventSubURL string `xml:"eventSubURL"`
}

type VendorXML struct {
	XMLName  xml.Name
	XMLAttrs []xml.Attr
	Value    any `xml:",innerxml"`
}

// MarshalXML generate XML output for VendorXML
// Example:
//
//	device.VendorXML = append(device.VendorXML, upnp.VendorXML{
//		XMLName: xml.Name{Local: "dlna:X_DLNADOC", Space: "urn:schemas-dlna-org:device-1-0"},
//		Value:   "DMS-1.50",
//	})
//
// will produce: <dlna:X_DLNADOC>DMS-1.50</dlna:X_DLNADOC>
// and to <root> will be added xmlns:dlna="urn:schemas-dlna-org:device-1-0"
func (v VendorXML) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = v.XMLName
	start.Attr = append(start.Attr, v.XMLAttrs...)
	// namespace added to DeviceDesc root element, cut it off from Vendor's root element
	if n := strings.Index(v.XMLName.Local, ":"); n > 0 {
		start.Name.Space = ""
	}
	return e.EncodeElement(v.Value, start)
}

type Device struct {
	// DeviceType Required. UPnP device type.
	DeviceType string `xml:"deviceType"`

	// FriendlyName Required. Short description for end user.
	// Should be localized. Should be < 64 characters.
	FriendlyName string `xml:"friendlyName"`

	// Manufacturer Required. Manufacturer's name.
	// May be localized. Should be < 64 characters.
	Manufacturer string `xml:"manufacturer"`

	// ManufacturerURL Optional. Website for Manufacturer.
	// May be localized. Single URL.
	ManufacturerURL string `xml:"manufacturerURL,omitempty"`

	// ModelDescription Recommended. Long description for end user.
	// Should be localized. Should be < 128 characters.
	ModelDescription string `xml:"modelDescription,omitempty"`

	// ModelName Required. Model name.
	// May be localized.  Should be < 32 characters.
	ModelName string `xml:"modelName"`

	// ModelNumber Recommended. Model number.
	// May be localized. Should be < 32 characters.
	ModelNumber string `xml:"modelNumber,omitempty"`

	// ModelURL Optional. Website for model.
	// May be localized. Single URL.
	ModelURL string `xml:"modelURL,omitempty"`

	// SerialNumber Recommended. Serial number.
	// May be localized. Should be < 64 characters.
	SerialNumber string `xml:"serialNumber,omitempty"`

	// UDN Required. Unique Device Name. Must begin with "uuid:".
	// Must be the same over time for a specific device instance
	UDN string

	// UPC Optional. Universal Product Code. 12-digit, all-numeric
	// code that identifies the consumer package. Single UPC.
	UPC string `xml:"UPC,omitempty"`

	// IconList Required if and only if device has one or more icons.
	IconList []Icon `xml:"iconList>icon"`

	// ServiceList Optional.
	ServiceList []*Service `xml:"serviceList>service"`

	// PresentationURL Recommended. URL to presentation for device.
	// May be relative to base URL. Single URL.
	PresentationURL string `xml:"presentationURL,omitempty"`

	// VendorXML Provide possibility to extend device Description.
	VendorXML []VendorXML
}

type DeviceDesc struct {
	// Required
	SpecVersion SpecVersion `xml:"specVersion"`

	// Device Required
	Device Device `xml:"device"`

	// URLBase Optional. Defines the base URL. If URLBase is empty or not given,
	// the base URL is the URL from which the device description was retrieved
	// (which is the preferred implementation; use of URLBase is no longer recommended). Single URL.
	URLBase string `xml:"URLBase,omitempty"`
}

// MarshalXML generate XML output for DeviceDesc
func (r DeviceDesc) MarshalXML(e *xml.Encoder, start xml.StartElement) error {

	var err error

	start.Name = xml.Name{Local: "root", Space: DeviceXMLNamespace}

	// collect unique vendor's namespaces and add them to root element
	spaces := map[string]string{}
	for _, item := range r.Device.VendorXML {
		n := strings.Index(item.XMLName.Local, ":")
		if n > 0 && item.XMLName.Space != "" {
			name := "xmlns:" + item.XMLName.Local[:n]
			space := item.XMLName.Space
			spaces[name] = space
		}
	}
	for name, space := range spaces {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Space: "", Local: name}, Value: space})
	}

	if err = e.EncodeToken(start); err != nil {
		return err
	}
	if err = e.EncodeElement(r.SpecVersion, xml.StartElement{Name: xml.Name{Local: "specVersion"}}); err != nil {
		return err
	}
	if err = e.EncodeElement(r.Device, xml.StartElement{Name: xml.Name{Local: "device"}}); err != nil {
		return err
	}
	if r.URLBase != "" {
		if err = e.EncodeElement(r.URLBase, xml.StartElement{Name: xml.Name{Local: "URLBase"}}); err != nil {
			return err
		}
	}
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}