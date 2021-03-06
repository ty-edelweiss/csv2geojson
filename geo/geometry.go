package geo

import (
	"crypto/sha1"
	"errors"
	"io"
	"strconv"

	"github.com/paulmach/go.geojson"
)

type Point []float64
type LineString [][]float64
type Polygon [][][]float64

func BuildPointCollection(longitude string, latitude string, columns []int, headers []string, records [][]string, limit int) *geojson.FeatureCollection {
	fc := geojson.NewFeatureCollection()

	for i, record := range records {
		report.ProgressTick(1.0)

		coord, err := ParseCoordinate(columns, record)
		if err != nil {
			logger.WithField("coordinate", coord).Warn(err)
			continue
		}

		feature := geojson.NewPointFeature(coord)

		properties := ParseProperties(headers, record, longitude, latitude)
		for key, property := range properties {
			feature.SetProperty(key, property)
		}

		if limit != 0 && limit == i {
			break
		}
		fc.AddFeature(feature)
	}

	return fc
}

func BuildLineStringCollection(longitude string, latitude string, index int, columns []int, headers []string, records [][]string, limit int) *geojson.FeatureCollection {
	fc := geojson.NewFeatureCollection()

	tmp := make(map[string]LineString)
	tmps := make(map[string]PropertyCollections)
	for _, record := range records {
		report.ProgressTick(0.5)

		coord, err := ParseCoordinate(columns, record)
		if err != nil {
			logger.WithField("coordinate", coord).Warn(err)
			continue
		}

		key := record[index]

		properties := ParseProperties(headers, record, longitude, latitude)

		tmp[key] = append(tmp[key], coord)

		if _, ok := tmps[key]; !ok {
			tmps[key] = PropertyCollections{}
		}
		tmps[key].AppendProperties(properties)
	}
	logger.WithField("length", len(tmp)).Debug("Features append computation order is following.")

	cnt := 0
	chunk := report.CreateChunk(len(tmp))
	for id, coords := range tmp {
		report.ProgressTick(chunk)

		feature := geojson.NewLineStringFeature(coords)

		feature.SetProperty("hash_", ParseHash(id))

		for key, pc := range tmps[id] {
			feature.SetProperty(key, pc)
		}

		if limit != 0 && limit == cnt {
			break
		}
		cnt++

		fc.AddFeature(feature)
	}

	return fc
}

func BuildPolygonCollection(longitude string, latitude string, index int, columns []int, headers []string, records [][]string, limit int) *geojson.FeatureCollection {
	fc := geojson.NewFeatureCollection()

	tmp := make(map[string]LineString)
	tmps := make(map[string]PropertyCollections)
	for _, record := range records {
		report.ProgressTick(0.5)

		coord, err := ParseCoordinate(columns, record)
		if err != nil {
			logger.WithField("coordinate", coord).Warn(err)
			continue
		}

		key := record[index]

		properties := ParseProperties(headers, record, longitude, latitude)

		tmp[key] = append(tmp[key], coord)

		if _, ok := tmps[key]; !ok {
			tmps[key] = PropertyCollections{}
		}
		tmps[key].AppendProperties(properties)
	}
	logger.WithField("length", len(tmp)).Debug("Features append computation order is following.")

	cnt := 0
	chunk := report.CreateChunk(len(tmp))
	for id, coords := range tmp {
		report.ProgressTick(chunk)

		polygon, err := ParsePolygon(coords)
		if err != nil {
			logger.WithField("key", id).Warn(err)
			continue
		}

		feature := geojson.NewPolygonFeature(polygon)

		feature.SetProperty("hash_", ParseHash(id))

		for key, prop := range tmps[id] {
			feature.SetProperty(key, prop)
		}

		if limit != 0 && limit == cnt {
			break
		}
		cnt++

		fc.AddFeature(feature)
	}

	return fc
}

func ParseCoordinate(columns []int, record []string) (Point, error) {
	if len(columns) == 1 {
		return []float64{}, errors.New("Coordinate format is invalid")
	}

	lon, err := strconv.ParseFloat(record[columns[0]], 64)
	if err != nil {
		return []float64{}, err
	}
	lat, err := strconv.ParseFloat(record[columns[1]], 64)
	if err != nil {
		return []float64{}, err
	}

	return Point{lon, lat}, nil
}

func ParsePolygon(lines ...LineString) (Polygon, error) {
	if len(lines) > 2 {
		return Polygon{}, errors.New("Polygon parse arguments is too many")
	}

	polygon := Polygon{}
	for _, line := range lines {
		if len(line) < 3 {
			return Polygon{}, errors.New("Coordinates format is invalid for polygon")
		}

		if line[len(line)-1][0] == line[0][0] && line[len(line)-1][1] == line[0][1] {
			polygon = append(polygon, line)
		} else {
			closed := append(line, line[0])
			polygon = append(polygon, closed)
		}
	}

	return polygon, nil
}

func ParseHash(key string) []byte {
	hash := sha1.New()

	io.WriteString(hash, key)

	buf := hash.Sum(nil)
	logger.WithField("hash", string(buf)).Info("Convert key to hash buffer done")

	return buf
}
