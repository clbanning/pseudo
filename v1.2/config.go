// config.go - set PseudoCtx from a config file with a "config":"pseudo" JSON object.

package pseudo

/* this is commented out unless you want to import github.com/clbanning/checkjson ...
import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/clbanning/checkjson"
)

// Config parses a file with a JSON object with Context settings - the
// JSON object in the file is identified by the key:value pair "config":"pseudo".
// 
// Example:
//	{
//	  "config":"pseudo",
//	  "lowestlabel": true,
//	  "fifobucket": true
//	}
func Config(file string) error {
	// read file into an array of JSON objects
	objs, err := checkjson.ReadJSONFile(file)
	if err != nil {
		return fmt.Errorf("config file: %s - %s", file, err.Error())
	}

	// get a JSON object that has "config":"pseudo" key:value pair
	type config struct {
		Config string
	}
	var ctxset bool // make sure just one pseudo config entry
	for n, obj := range objs {
		c := new(config)
		// unmarshal the object - and try and retrule a meaningful error
		if err := json.Unmarshal(obj, c); err != nil {
			return fmt.Errorf("parsing config file: %s entry: %d - %s",
				file, n+1, checkjson.ResolveJSONError(obj, err).Error())
		}
		switch strings.ToLower(c.Config) {
		case "pseudo":
			if ctxset {
				return fmt.Errorf("duplicate 'pseudo' entry in config file: %s entry: %d", file, n)
			}
			if err := checkjson.Validate(obj, PseudoCtx); err != nil {
				return fmt.Errorf("checking pseudo config JSON object: %s", err)
			}
			if err := json.Unmarshal(obj, &PseudoCtx); err != nil {
				return fmt.Errorf("config file: %s - %s", file, err)
			}
			ctxset = true
		default:
			// return fmt.Errorf("unknown config option in config file: %s entry: %d", file, n+1)
			// for now, just ignore stuff we're not interested in
		}
	}
	if !ctxset {
		return fmt.Errorf("no pseudo config object in %s", file)
	}

	initGlobals()
	return nil
}
*/
