package dlna

import (
	"fmt"
	"strings"
)

type ContentFeatures struct {
	pn    string
	op    string
	ci    string
	flags string
}

func NewContentFeatures() *ContentFeatures {
	return &ContentFeatures{}
}

func NewThumbContentFeatures() *ContentFeatures {
	return new(ContentFeatures).
		Profile("JPEG_TN").
		Flags("00f00000000000000000000000000000")
}

func NewMediaContentFeatures(profile ...string) *ContentFeatures {
	if len(profile) == 0 {
		profile = append(profile, "")
	}
	return new(ContentFeatures).
		Profile(profile[0]).
		Seek(false, true).
		Flags("01700000000000000000000000000000")
}

func NewLiveStreamContentFeatures(profile ...string) *ContentFeatures {
	if len(profile) == 0 {
		profile = append(profile, "")
	}
	return new(ContentFeatures).
		Profile(profile[0]).
		Seek(false, true).
		Flags("8D500000000000000000000000000000")
}

func (cf *ContentFeatures) Profile(profileName string) *ContentFeatures {
	if profileName == "" {
		cf.pn = ""
	} else {
		cf.pn = fmt.Sprintf("DLNA.ORG_PN=%s", profileName)
	}
	return cf
}

func (cf *ContentFeatures) Seek(supportTimeSeek bool, supportByteRange bool) *ContentFeatures {
	cf.op = fmt.Sprintf("DLNA.ORG_OP=%s%s", BinaryStr(supportTimeSeek), BinaryStr(supportByteRange))
	return cf
}

func (cf *ContentFeatures) Transcode(transcode bool) *ContentFeatures {
	cf.ci = fmt.Sprintf("DLNA.ORG_CI=%s", BinaryStr(transcode))
	return cf
}

func (cf *ContentFeatures) Flags(flags string) *ContentFeatures {
	cf.flags = fmt.Sprintf("DLNA.ORG_FLAGS=%s", flags)
	return cf
}

func (cf *ContentFeatures) String() string {
	args := make([]string, 0)
	if cf.pn != "" {
		args = append(args, cf.pn)
	}
	if cf.op != "" {
		args = append(args, cf.op)
	}
	if cf.ci != "" {
		args = append(args, cf.ci)
	}
	if cf.flags != "" {
		args = append(args, cf.flags)
	}
	return strings.Join(args, ";")
}

func BinaryStr(v bool) string {
	if v {
		return "1"
	}
	return "0"
}
