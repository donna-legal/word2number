package resources

import (
	"fmt"
	"path"

	"github.com/labstack/gommon/log"
	y2 "gopkg.in/yaml.v2"
)

type craymap map[string]map[string][]interface{}

var arraymap craymap

func init() {
	arraymap = make(craymap)
	readFolders(arraymap, "resources")
}

//ArrayMap returns an array of a map of values
func ArrayMap(locale, key string) (out []map[string]string) {
	if _, ok := arraymap[locale]; ok {
		for _, val := range arraymap[locale][key] {
			if imap, ok := val.(map[interface{}]interface{}); ok && val != nil {
				m := make(map[string]string)
				for k, v := range imap {
					m[fmt.Sprint(k)] = fmt.Sprint(v)
				}
				out = append(out, m)
			}
		}
	}
	return
}

func readFolders(maap craymap, folders ...string) {
	for _, folder := range folders {
		filenames, err := AssetDir(folder)
		if err != nil {
			continue
		}
		for _, filename := range filenames {
			bytes, err := Asset(path.Join(folder, filename))
			if err != nil {
				log.Error(err)
				continue
			}
			m := make(craymap)
			err = y2.Unmarshal(bytes, &m)
			if err != nil {
				log.Error(err)
				continue
			}
			for locale, m2 := range m {
				if oldm, ok := maap[locale]; ok {
					for k, v := range m2 {
						oldm[k] = v
					}
				} else {
					maap[locale] = m2
				}
			}
		}
	}
}
