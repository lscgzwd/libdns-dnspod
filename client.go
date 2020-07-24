package dnspod

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/libdns/libdns"
	d "github.com/nrdcg/dnspod-go"
)

// Client ...
type Client struct {
	client     *d.Client
	mutex      sync.Mutex
	domainList []d.Domain
}

func (p *Provider) getClient() error {
	if p.client == nil {
		params := d.CommonParams{LoginToken: p.APIToken, Format: "json"}
		p.client = d.NewClient(params)
	}

	return nil
}
func (p *Provider) getDomains() ([]d.Domain, error) {
	if len(p.domainList) > 0 {
		return p.domainList, nil
	}
	domains, _, err := p.client.Domains.List()
	if nil != err {
		return p.domainList, err
	}
	p.domainList = domains
	return p.domainList, nil
}
func (p *Provider) getDomainIDByDomainName(domainName string) (string, error) {
	domains, err := p.getDomains()
	if nil != err {
		return "", err
	}
	domainName = strings.Trim(domainName, ".")
	for _, domain := range domains {
		if domain.Name == domainName {
			return string(domain.ID), nil
		}
	}
	return "", fmt.Errorf("Domain %s not found in your dnspod account", domainName)
}

func (p *Provider) getDNSEntries(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.getClient()

	var records []libdns.Record
	domainID, err := p.getDomainIDByDomainName(zone)
	if nil != err {
		return records, err
	}
	//todo now can only return 100 records
	reqRecords, _, err := p.client.Records.List(string(domainID), "")
	if err != nil {
		return records, err
	}

	for _, entry := range reqRecords {
		ttl, _ := strconv.ParseInt(entry.TTL, 10, 64)
		record := libdns.Record{
			Name:  entry.Name,
			Value: entry.Value,
			Type:  entry.Type,
			TTL:   time.Duration(ttl) * time.Second,
			ID:    entry.ID,
		}
		records = append(records, record)
	}

	return records, nil
}

func (p *Provider) addDNSEntry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.getClient()

	entry := d.Record{
		Name:  record.Name,
		Value: record.Value,
		Type:  record.Type,
		Line:  "默认",
		TTL:   strconv.Itoa(int(record.TTL.Seconds())),
	}
	domainID, err := p.getDomainIDByDomainName(zone)
	if nil != err {
		return record, err
	}
	rec, _, err := p.client.Records.Create(domainID, entry)
	if err != nil {
		return record, err
	}
	record.ID = rec.ID

	return record, nil
}

func (p *Provider) removeDNSEntry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.getClient()

	domainID, err := p.getDomainIDByDomainName(zone)
	if nil != err {
		return record, err
	}
	_, err = p.client.Records.Delete(domainID, record.ID)
	if err != nil {
		return record, err
	}

	return record, nil
}

func (p *Provider) updateDNSEntry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.getClient()

	entry := d.Record{
		Name:  record.Name,
		Value: record.Value,
		Type:  record.Type,
		Line:  "默认",
		TTL:   strconv.Itoa(int(record.TTL.Seconds())),
	}
	domainID, err := p.getDomainIDByDomainName(zone)
	if nil != err {
		return record, err
	}
	_, _, err = p.client.Records.Update(domainID, record.ID, entry)
	if err != nil {
		return record, err
	}

	return record, nil
}
