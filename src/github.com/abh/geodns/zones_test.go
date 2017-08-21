package main

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/miekg/dns"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the gotest runner.
func Test(t *testing.T) { TestingT(t) }

type ConfigSuite struct {
	srv   *Server
	zones Zones
}

var _ = Suite(&ConfigSuite{})

func (s *ConfigSuite) SetUpSuite(c *C) {
	s.zones = make(Zones)
	lastRead = map[string]*ZoneReadRecord{}
	s.srv = &Server{}
	s.srv.zonesReadDir("dns", s.zones)
}

func (s *ConfigSuite) TestReadConfigs(c *C) {
	// Just check that example.com and test.example.org loaded, too.
	c.Check(s.zones["example.com"].Origin, Equals, "example.com")
	c.Check(s.zones["test.example.org"].Origin, Equals, "test.example.org")
	if s.zones["test.example.org"].Options.Serial == 0 {
		c.Log("Serial number is 0, should be set by file timestamp")
		c.Fail()
	}

	// The real tests are in test.example.com so we have a place
	// to make nutty configuration entries
	tz := s.zones["test.example.com"]

	// test.example.com was loaded
	c.Check(tz.Origin, Equals, "test.example.com")

	c.Check(tz.Options.MaxHosts, Equals, 2)
	c.Check(tz.Options.Contact, Equals, "support.bitnames.com")
	c.Check(tz.Options.Targeting.String(), Equals, "@ continent country regiongroup region asn ip")

	// Got logging option
	c.Check(tz.Logging.StatHat, Equals, true)

	c.Check(tz.Labels["weight"].MaxHosts, Equals, 1)

	/* test different cname targets */
	c.Check(tz.Labels["www"].
		firstRR(dns.TypeCNAME).(*dns.CNAME).
		Target, Equals, "geo.bitnames.com.")

	c.Check(tz.Labels["www-cname"].
		firstRR(dns.TypeCNAME).(*dns.CNAME).
		Target, Equals, "bar.test.example.com.")

	c.Check(tz.Labels["www-alias"].
		firstRR(dns.TypeMF).(*dns.MF).
		Mf, Equals, "www")

	// The header name should just have a dot-prefix
	c.Check(tz.Labels[""].Records[dns.TypeNS][0].RR.(*dns.NS).Hdr.Name, Equals, "test.example.com.")

}

func (s *ConfigSuite) TestRemoveConfig(c *C) {
	// restore the dns.Mux
	defer s.srv.zonesReadDir("dns", s.zones)

	dir, err := ioutil.TempDir("", "geodns-test.")
	if err != nil {
		c.Fail()
	}
	defer os.RemoveAll(dir)

	_, err = CopyFile(c, "dns/test.example.org.json", dir+"/test.example.org.json")
	if err != nil {
		c.Log(err)
		c.Fail()
	}
	_, err = CopyFile(c, "dns/test.example.org.json", dir+"/test2.example.org.json")
	if err != nil {
		c.Log(err)
		c.Fail()
	}

	err = ioutil.WriteFile(dir+"/invalid.example.org.json", []byte("not-json"), 0644)
	if err != nil {
		c.Log(err)
		c.Fail()
	}

	s.srv.zonesReadDir(dir, s.zones)
	c.Check(s.zones["test.example.org"].Origin, Equals, "test.example.org")
	c.Check(s.zones["test2.example.org"].Origin, Equals, "test2.example.org")

	os.Remove(dir + "/test2.example.org.json")
	os.Remove(dir + "/invalid.example.org.json")

	s.srv.zonesReadDir(dir, s.zones)
	c.Check(s.zones["test.example.org"].Origin, Equals, "test.example.org")
	_, ok := s.zones["test2.example.org"]
	c.Check(ok, Equals, false)
}

func CopyFile(c *C, src, dst string) (int64, error) {
	sf, err := os.Open(src)
	if err != nil {
		c.Log("Could not copy", src, "to", dst, "because", err)
		c.Fail()
		return 0, err
	}
	defer sf.Close()
	df, err := os.Create(dst)
	if err != nil {
		c.Log("Could not copy", src, "to", dst, "because", err)
		c.Fail()
		return 0, err
	}
	defer df.Close()
	return io.Copy(df, sf)
}
