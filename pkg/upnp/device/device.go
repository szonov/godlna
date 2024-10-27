package device

import (
	"crypto/md5"
	"encoding/xml"
	"fmt"
	"strings"
)

const (
	XMLNamespace = "urn:schemas-upnp-org:device-1-0"
)

// reference https://upnp.org/specs/arch/UPnP-arch-DeviceArchitecture-v1.0-20080424.pdf
// on the page 26 there is XML listing and then detailed description

var Version = SpecVersion{
	Major: 1,
	Minor: 0,
}

type SpecVersion struct {
	// Required. Major version of the UPnP Device Architecture. Must be 1.
	Major uint `xml:"major"`
	// Required. Minor version of the UPnP Device Architecture.  Must be 0 in devices that implement UDA version 1.0.
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
	// URL Required. Pointer to icon image. May be relative to base URL. Single URL.
	URL string `xml:"url"`
}

type Service struct {
	XMLName xml.Name `xml:"service"`
	// ServiceType Required. UPnP service type.  Must not contain a hash character (#, 23 Hex in UTF-8).
	ServiceType string `xml:"serviceType"`
	// ServiceId Required. Service identifier. Must be unique within this device description.
	ServiceId string `xml:"serviceId"`
	// SCPDURL Required. URL for service description (Service Control Protocol Definition URL). Single URL.
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

// VendorXMLTag gives possibility to add additional (vendor depended) tags to device
type VendorXMLTag struct {
	XMLName  xml.Name
	XMLAttrs []xml.Attr
	Value    any `xml:",innerxml"`
}

// VendorXML gives possibility to add additional (vendor depended) tags to device
type VendorXML []VendorXMLTag

// MarshalXML generate XML output for VendorXML
// Example:
//
//	device.VendorXML = append(device.VendorXML, device.VendorXML{
//		XMLName: xml.Name{Local: "dlna:X_DLNADOC", Space: "urn:schemas-dlna-org:device-1-0"},
//		Value:   "DMS-1.50",
//	})
//
// will produce: <dlna:X_DLNADOC>DMS-1.50</dlna:X_DLNADOC>
// and to <root> will be added xmlns:dlna="urn:schemas-dlna-org:device-1-0"
func (v VendorXMLTag) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = v.XMLName
	start.Attr = append(start.Attr, v.XMLAttrs...)
	// namespace added to Description root element, cut it off from Vendor's root element
	if n := strings.Index(v.XMLName.Local, ":"); n > 0 {
		start.Name.Space = ""
	}
	return e.EncodeElement(v.Value, start)
}

func (v VendorXML) Add(prefix string, namespace string, tags ...[2]string) VendorXML {
	ret := v
	for _, t := range tags {
		ret = append(ret, VendorXMLTag{
			XMLName: xml.Name{Local: fmt.Sprintf("%s:%s", prefix, t[0]), Space: namespace},
			Value:   t[1],
		})
	}
	return ret
}

func NewVendorXML() VendorXML {
	return make(VendorXML, 0)
}

func VendorValue(name, value string) [2]string {
	return [2]string{name, value}
}

type Device struct {
	// DeviceType Required. UPnP device type.
	DeviceType string `xml:"deviceType"`
	// FriendlyName Required. Short description for end user.  Should be localized. Should be < 64 characters.
	FriendlyName string `xml:"friendlyName"`
	// Manufacturer Required. Manufacturer's name. May be localized. Should be < 64 characters.
	Manufacturer string `xml:"manufacturer"`
	// ManufacturerURL Optional. Website for Manufacturer. May be localized. Single URL.
	ManufacturerURL string `xml:"manufacturerURL,omitempty"`
	// ModelDescription Recommended. Long description for end user. Should be localized. Should be < 128 characters.
	ModelDescription string `xml:"modelDescription,omitempty"`
	// ModelName Required. Model name. May be localized.  Should be < 32 characters.
	ModelName string `xml:"modelName"`
	// ModelNumber Recommended. Model number. May be localized. Should be < 32 characters.
	ModelNumber string `xml:"modelNumber,omitempty"`
	// ModelURL Optional. Website for model. May be localized. Single URL.
	ModelURL string `xml:"modelURL,omitempty"`
	// SerialNumber Recommended. Serial number. May be localized. Should be < 64 characters.
	SerialNumber string `xml:"serialNumber,omitempty"`
	// UDN Required. Unique Device Name. Must begin with "uuid:". Must be the same over time for a specific device instance
	UDN string
	// UPC Optional. Universal Product Code. 12-digit, all-numeric code that identifies the consumer package. Single UPC.
	UPC string `xml:"UPC,omitempty"`
	// IconList Required if and only if device has one or more icons.
	IconList []Icon `xml:"iconList>icon"`
	// ServiceList Optional.
	ServiceList []*Service `xml:"serviceList>service"`
	// PresentationURL Recommended. URL to presentation for device. May be relative to base URL. Single URL.
	PresentationURL string `xml:"presentationURL,omitempty"`
	// VendorXML Provide possibility to extend device Description.
	VendorXML VendorXML
}

type Description struct {
	// Required
	SpecVersion SpecVersion `xml:"specVersion"`
	// Device Required
	Device *Device `xml:"device"`
	// URLBase Optional. Defines the base URL. If URLBase is empty or not given,
	// the base URL is the URL from which the device description was retrieved
	// (which is the preferred implementation; use of URLBase is no longer recommended). Single URL.
	URLBase string `xml:"URLBase,omitempty"`
	// Location Url, on which description will be available, used as 'Location' header for SSDP
	Location string `xml:"-"`
}

// MarshalXML generate XML output for Description
func (r *Description) MarshalXML(e *xml.Encoder, start xml.StartElement) error {

	var err error

	start.Name = xml.Name{Local: "root", Space: XMLNamespace}

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
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: name}, Value: space})
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

// NewUDN is a helper to create new UDN for device
func NewUDN(unique string) string {
	hash := md5.Sum([]byte(unique))
	return fmt.Sprintf("uuid:%x-%x-%x-%x-%x", hash[:4], hash[4:6], hash[6:8], hash[8:10], hash[10:16])
}
