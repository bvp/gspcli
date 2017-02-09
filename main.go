package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"text/template"
)

const (
	provisionFile = "provision.xml"
)

var (
	outPath     string
	baseConfig  string
	devicesFile string
)

// Device for mapping
type Device struct {
	Mac      string
	User     string
	Password string
	AuthID   string
}

// Setting struct
type Setting struct {
	ID       string
	Value    string
	Comments []string
}

// Config struct
type Config struct {
	Dev      Device
	Settings []Setting
}

func writeFile(fn string, content string) {
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		errmd := os.MkdirAll(outPath, 0755)
		if errmd != nil {
			log.Fatalf("Unable to create output directory: %s\n", errmd.Error())
		}
	}
	p := path.Join(outPath, fn)
	err := ioutil.WriteFile(p, []byte(content), 0644)
	log.Printf("The file is written in %s\n", p)
	if err != nil {
		log.Fatalf("Unable to write file: %s\n", err.Error())
	}
}

func parseTemplate(c Config) (filename string, content string) {
	var doc bytes.Buffer
	t, _ := template.New("Provisioning template").ParseFiles(provisionFile)
	t.ExecuteTemplate(&doc, provisionFile, c)

	return "cfg" + c.Dev.Mac + ".xml", doc.String()
}

func parseConfig(device Device) (Config, error) {
	var (
		config  Config
		setting Setting
	)

	config.Dev.Mac = device.Mac
	config.Dev.User = device.User
	config.Dev.Password = device.Password
	config.Dev.AuthID = device.AuthID

	file, err := os.Open(baseConfig)
	defer file.Close()

	if err == nil {
		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			if strings.HasPrefix(scanner.Text(), "#") {
				setting.Comments = append(setting.Comments, scanner.Text())
			} else if strings.Contains(scanner.Text(), "=") {
				sd := strings.Split(scanner.Text(), "=")
				setting.ID = strings.TrimSpace(sd[0])
				switch setting.ID {
				case "P34", "P4120", "P3120":
					setting.Value = device.Password
				case "P35", "P4060", "P3060":
					setting.Value = device.User
				case "P36", "P4090", "P3090":
					setting.Value = device.AuthID
				default:
					setting.Value = strings.TrimSpace(sd[1])
				}
				config.Settings = append(config.Settings, setting)
				setting = Setting{}
			}
		}

		if err := scanner.Err(); err != nil {
			log.Fatalf("An error occurred while reading the configuration file: %s\n", err.Error())
		}
		return config, nil
	}
	log.Fatalf("Was unable to open the configuration file: %s\n", err.Error())
	return config, err
}

func loadCSV(file string) ([]Device, error) {
	var csvRecords []Device

	f, e := os.Open(file)
	if e == nil {
		r := csv.NewReader(bufio.NewReader(f))

		for {
			record, err := r.Read()
			if err == io.EOF {
				break
			}
			if strings.HasPrefix(strings.ToLower(record[0]), "000b82") {
				csvRecords = append(csvRecords, Device{Mac: record[0], User: record[1], Password: record[2], AuthID: record[3]})
			} else {
				log.Printf("Notice: The Mac \"%s\" is invalid for not starting with \"000B82\", please check the CSV file", record[0])
			}
		}
		return csvRecords, nil
	}
	log.Fatalf("Open devices file failed: %s\n", e.Error())
	return nil, e
}

func main() {
	flag.StringVar(&baseConfig, "c", "GsBaseConfig.txt", "Location base configuration file")
	flag.StringVar(&devicesFile, "d", "MAC.csv", "Location devices configuration file")
	flag.StringVar(&outPath, "o", "provisioning/GrandStream", "Output directory")
	flag.Parse()

	if devices, e := loadCSV(devicesFile); e == nil {
		for _, device := range devices {
			if c, e := parseConfig(device); e == nil {
				writeFile(parseTemplate(c))
			}
		}
	} else {
		log.Fatalf("ERROR: %s\n", e)
	}
}
