package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/cfmobile/gopivnet/resource"
)

type Api interface {
	GetLatestProductFile(productName string, fileType string) (*resource.ProductFile, error)
	GetProductFileForVersion(productName, version string, fileType string) (*resource.ProductFile, error)
	GetVersionsForProduct(productName string) ([]string, error)
	Download(productFile *resource.ProductFile, fileName string) error
}

type PivnetApi struct {
	Requester resource.ReleaseRequester
}

func New(token string) Api {
	return &PivnetApi{
		Requester: resource.NewRequester("https://network.pivotal.io", token),
	}
}

func (p *PivnetApi) GetLatestProductFile(productName string, fileType string) (*resource.ProductFile, error) {
	if productName == "" {
		return nil, errors.New("Must specify a product name")
	}

	prod, err := p.Requester.GetProduct(productName)
	if err != nil {
		return nil, err
	}

	productFiles, err := p.Requester.GetProductFiles(prod.Releases[0])
	if err != nil {
		return nil, err
	}

	pivotalProduct := getPivotalProduct(productFiles, fileType)
	if pivotalProduct == nil {
		return nil, errors.New("Unable to find a pivotal product")
	}

	return pivotalProduct, nil
}

func getPivotalProduct(productFiles *resource.ProductFiles, fileType string) *resource.ProductFile {
	for index, productFile := range productFiles.Files {
		if strings.Contains(productFile.AwsObjectKey, "."+fileType) {
			return &productFiles.Files[index]
		}
	}

	return nil
}

func (p *PivnetApi) GetProductFileForVersion(productName, version string, fileType string) (*resource.ProductFile, error) {
	if productName == "" {
		return nil, errors.New("Must specify a product name")
	}

	if version == "" {
		return nil, errors.New("Must specify a product version")
	}

	prod, err := p.Requester.GetProduct(productName)
	if err != nil {
		return nil, err
	}

	matchingRelease := getReleaseForVersion(prod, version)
	if matchingRelease == nil {
		return nil, errors.New("Specified version not found")
	}

	productFiles, err := p.Requester.GetProductFiles(*matchingRelease)
	if err != nil {
		return nil, err
	}

	pivotalProduct := getPivotalProduct(productFiles, fileType)
	if pivotalProduct == nil {
		return nil, errors.New("Unable to find a pivotal product")
	}

	return pivotalProduct, nil
}

func getReleaseForVersion(product *resource.Product, version string) *resource.Release {
	for index, release := range product.Releases {
		if release.Version == version {
			return &product.Releases[index]
		}
	}

	return nil
}

func (p *PivnetApi) GetVersionsForProduct(productName string) ([]string, error) {

	if len(productName) == 0 {
		return []string{}, errors.New("Product name was empty")
	}

	product, err := p.Requester.GetProduct(productName)

	if err != nil {
		return []string{}, err
	}

	var versions []string
	for _, release := range product.Releases {
		versions = append(versions, release.Version)
	}
	return versions, nil
}

func (p *PivnetApi) Download(productFile *resource.ProductFile, fileName string) error {
	if productFile == nil {
		return errors.New("Nil product passed in")
	}

	url, err := p.Requester.GetProductDownloadUrl(productFile)
	if err != nil {
		return err
	}

	return download(url, fileName)
}

func download(url, fileName string) error {
	out, err := os.Create(fileName)
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	fmt.Printf("Wrote %d bytes to \"%s\"\n", n, fileName)
	return nil
}
