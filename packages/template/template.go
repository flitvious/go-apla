// Apla Software includes an integrated development
// environment with a multi-level system for the management
// of access rights to data, interfaces, and Smart contracts. The
// technical characteristics of the Apla Software are indicated in
// Apla Technical Paper.
//
// Apla Users are granted a permission to deal in the Apla
// Software without restrictions, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of Apla Software, and to permit persons
// to whom Apla Software is furnished to do so, subject to the
// following conditions:
// * the copyright notice of GenesisKernel and EGAAS S.A.
// and this permission notice shall be included in all copies or
// substantial portions of the software;
// * a result of the dealing in Apla Software cannot be
// implemented outside of the Apla Platform environment.
//
// THE APLA SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY
// OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED
// TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A
// PARTICULAR PURPOSE, ERROR FREE AND NONINFRINGEMENT. IN
// NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR
// THE USE OR OTHER DEALINGS IN THE APLA SOFTWARE.

package template

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/AplaProject/go-apla/packages/consts"
	"github.com/AplaProject/go-apla/packages/converter"
	"github.com/AplaProject/go-apla/packages/language"
	"github.com/AplaProject/go-apla/packages/smart"
	"github.com/AplaProject/go-apla/packages/utils/tx"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

const (
	tagText = `text`
	tagData = `data`
	maxDeep = 16
)

type node struct {
	Tag      string                 `json:"tag"`
	Attr     map[string]interface{} `json:"attr,omitempty"`
	Text     string                 `json:"text,omitempty"`
	Children []*node                `json:"children,omitempty"`
	Tail     []*node                `json:"tail,omitempty"`
}

// Source describes dbfind or data source
type Source struct {
	Columns *[]string
	Data    *[][]string
}

// Workspace represents a workspace of executable template
type Workspace struct {
	Sources       *map[string]Source
	Vars          *map[string]string
	SmartContract *smart.SmartContract
	Timeout       *bool
}

// SetSource sets source to workspace
func (w *Workspace) SetSource(name string, source *Source) {
	if w.Sources == nil {
		sources := make(map[string]Source)
		w.Sources = &sources
	}
	(*w.Sources)[name] = *source
}

type parFunc struct {
	Owner     *node
	Node      *node
	Workspace *Workspace
	Pars      *map[string]string
	RawPars   *map[string]string
	Tails     *[]*[][]rune
}

type nodeFunc func(par parFunc) string

type tplFunc struct {
	Func   nodeFunc // process function
	Full   nodeFunc // full process function
	Tag    string   // HTML tag
	Params string   // names of parameters
}

type tailInfo struct {
	tplFunc
	Last bool
}

type forTails struct {
	Tails map[string]tailInfo
}

func newSource(par parFunc) {
	if par.Node.Attr[`source`] == nil {
		return
	}
	par.Workspace.SetSource(par.Node.Attr[`source`].(string), &Source{
		Columns: par.Node.Attr[`columns`].(*[]string),
		Data:    par.Node.Attr[`data`].(*[][]string),
	})
}

func setAttr(par parFunc, name string) {
	if len((*par.Pars)[name]) > 0 {
		par.Node.Attr[strings.ToLower(name)] = (*par.Pars)[name]
	}
}

func setAllAttr(par parFunc) {
	for key, v := range *par.Pars {
		if key == `Params` || key == `PageParams` {
			imap := make(map[string]interface{})
			re := regexp.MustCompile(`(?is)(.*)\((.*)\)`)
			parList := make([]string, 0, 10)
			curPar := make([]rune, 0, 256)
			stack := make([]rune, 0, 256)
			for _, ch := range v {
				switch ch {
				case '"':
					if len(stack) > 0 && stack[len(stack)-1] == '"' {
						stack = stack[:len(stack)-1]
					} else {
						stack = append(stack, '"')
					}
				case '(':
					stack = append(stack, ')')
				case '{':
					stack = append(stack, '}')
				case '[':
					stack = append(stack, ']')
				case ')', '}', ']':
					if len(stack) > 0 && stack[len(stack)-1] == ch {
						stack = stack[:len(stack)-1]
					}
				case ',':
					if len(stack) == 0 {
						parList = append(parList, string(curPar))
						curPar = curPar[:0]
						continue
					}
				}
				curPar = append(curPar, ch)
			}
			if len(curPar) > 0 {
				parList = append(parList, string(curPar))
			}
			for _, parval := range parList {
				parval = strings.TrimSpace(parval)
				if len(parval) > 0 {
					if off := strings.IndexByte(parval, '='); off == -1 {
						imap[parval] = map[string]interface{}{
							`type`: `text`, `text`: parval}
					} else {
						val := strings.TrimSpace(parval[off+1:])
						if ret := re.FindStringSubmatch(val); len(ret) == 3 {
							plist := strings.Split(ret[2], `,`)
							for i, ilist := range plist {
								plist[i] = strings.TrimSpace(ilist)
							}
							imap[strings.TrimSpace(parval[:off])] = map[string]interface{}{
								`type`: ret[1], `params`: plist}
						} else {
							imap[strings.TrimSpace(parval[:off])] = map[string]interface{}{
								`type`: `text`, `text`: val}
						}
					}
				}
			}
			if len(imap) > 0 {
				par.Node.Attr[strings.ToLower(key)] = imap
			}
		} else if key != `Body` && (key != `Data` || (*par.Workspace.Vars)[`_full`] == `1`) &&
			len(v) > 0 {
			par.Node.Attr[strings.ToLower(key)] = v
		}
	}
	for key := range *par.Pars {
		if len(key) == 0 || key[0] != '@' {
			continue
		}

		key = strings.ToLower(key[1:])
		if par.Node.Attr[key] == nil {
			continue
		}
		par.Node.Attr[key] = processToText(par, par.Node.Attr[key].(string))
	}
}

func processToText(par parFunc, input string) (out string) {
	root := node{}
	process(input, &root, par.Workspace)
	for _, item := range root.Children {
		if item.Tag == `text` {
			out += item.Text
		}
	}
	return
}

func ifValue(val string, workspace *Workspace) bool {
	var sep string

	val = parseArg(val, workspace)

	if strings.Index(val, `;base64`) < 0 {
		for _, item := range []string{`==`, `!=`, `<=`, `>=`, `<`, `>`} {
			if strings.Index(val, item) >= 0 {
				sep = item
				break
			}
		}
	}
	cond := []string{val}
	if len(sep) > 0 {
		cond = strings.SplitN(val, sep, 2)
		cond[0], cond[1] = macro(strings.Trim(cond[0], `"`), workspace.Vars),
			macro(strings.Trim(cond[1], `"`), workspace.Vars)
	} else {
		val = macro(val, workspace.Vars)
	}
	switch sep {
	case ``:
		return len(val) > 0 && val != `0` && val != `false`
	case `==`:
		return len(cond) == 2 && strings.TrimSpace(cond[0]) == strings.TrimSpace(cond[1])
	case `!=`:
		return len(cond) == 2 && strings.TrimSpace(cond[0]) != strings.TrimSpace(cond[1])
	case `>`, `<`, `<=`, `>=`:
		ret0, _ := decimal.NewFromString(strings.TrimSpace(cond[0]))
		ret1, _ := decimal.NewFromString(strings.TrimSpace(cond[1]))
		if len(cond) == 2 {
			var bin bool
			if sep == `>` || sep == `<=` {
				bin = ret0.Cmp(ret1) > 0
			} else {
				bin = ret0.Cmp(ret1) < 0
			}
			if sep == `<=` || sep == `>=` {
				bin = !bin
			}
			return bin
		}
	}
	return false
}

func replace(input string, level *[]string, vars *map[string]string) string {
	if len(input) == 0 {
		return input
	}
	result := make([]rune, 0, utf8.RuneCountInString(input))
	isName := false
	name := make([]rune, 0, 128)
	syschar := '#'
	clearname := func() {
		result = append(append(result, syschar), name...)
		isName = false
		name = name[:0]
	}
	for _, r := range input {
		if r != syschar {
			if isName {
				name = append(name, r)
				if len(name) > 64 || r <= ' ' {
					clearname()
				}
			} else {
				result = append(result, r)
			}
			continue
		}
		if isName {
			if value, ok := (*vars)[string(name)]; ok {
				var loop bool
				if len(*level) < maxDeep {
					for _, item := range *level {
						if item == string(name) {
							loop = true
							break
						}
					}
				} else {
					loop = true
				}
				if !loop {
					*level = append(*level, string(name))
					value = replace(value, level, vars)
					*level = (*level)[:len(*level)-1]
					result = append(result, []rune(value)...)
				} else {
					result = append(append(result, syschar), append(name, syschar)...)
				}
				isName = false
			} else {
				result = append(append(result, syschar), name...)
			}
			name = name[:0]
		} else {
			isName = true
		}
	}
	if isName {
		result = append(append(result, syschar), name...)
	}
	return string(result)
}

func macro(input string, vars *map[string]string) string {
	if (*vars)[`_full`] == `1` || strings.IndexByte(input, '#') == -1 {
		return input
	}
	return macroReplace(input, vars)
}

func macroReplace(input string, vars *map[string]string) string {
	level := make([]string, 0, maxDeep)
	return replace(input, &level, vars)
}

func appendText(owner *node, text string) {
	if len(strings.TrimSpace(text)) == 0 {
		return
	}
	if len(text) > 0 {
		owner.Children = append(owner.Children, &node{Tag: tagText, Text: text})
	}
}

func callFunc(curFunc *tplFunc, owner *node, workspace *Workspace, params *[][]rune, tailpars *[]*[][]rune) {
	var (
		out     string
		curNode node
	)
	pars := make(map[string]string)
	parFunc := parFunc{
		Workspace: workspace,
	}
	if *workspace.Timeout {
		return
	}
	trim := func(input string, quotes bool) string {
		result := strings.Trim(input, "\t\r\n ")
		if quotes && len(result) > 0 {
			for _, ch := range "\"`" {
				if rune(result[0]) == ch {
					result = strings.Trim(result, string([]rune{ch}))
					break
				}
			}
		}
		return result
	}
	if curFunc.Params == `*` {
		for i, v := range *params {
			val := strings.TrimSpace(string(v))
			off := strings.IndexByte(val, ':')
			if off != -1 {
				pars[val[:off]] = macro(trim(val[off+1:], true), workspace.Vars)
			} else {
				pars[strconv.Itoa(i)] = val
			}
		}
	} else {
		for i, v := range strings.Split(curFunc.Params, `,`) {
			if i < len(*params) {
				val := strings.TrimSpace(string((*params)[i]))
				off := strings.IndexByte(val, ':')
				if off != -1 && strings.Contains(curFunc.Params, `#`+val[:off]) {
					pars[`#`+val[:off]] = trim(val[off+1:], val[:off] != `Data`)
				} else if off != -1 && strings.Contains(curFunc.Params, val[:off]) {
					pars[val[:off]] = trim(val[off+1:], val[:off] != `Data`)
				} else {
					pars[v] = val
				}
			} else if _, ok := pars[v]; !ok {
				pars[v] = ``
			}
		}
	}
	state := int(converter.StrToInt64((*workspace.Vars)[`ecosystem_id`]))
	if (*workspace.Vars)[`_full`] != `1` {
		for i, v := range pars {
			pars[i] = language.LangMacro(v, state, (*workspace.Vars)[`lang`])
			if pars[i] != v {
				if parFunc.RawPars == nil {
					rawpars := make(map[string]string)
					parFunc.RawPars = &rawpars
				}
				(*parFunc.RawPars)[i] = v
			}
		}
	}
	if len(curFunc.Tag) > 0 {
		curNode.Tag = curFunc.Tag
		curNode.Attr = make(map[string]interface{})
		if len(pars[`Body`]) > 0 && curFunc.Tag != `custom` {
			if (curFunc.Tag != `if` && curFunc.Tag != `elseif`) || (*workspace.Vars)[`_full`] == `1` {
				process(pars[`Body`], &curNode, workspace)
			}
		}
		parFunc.Owner = owner
		parFunc.Node = &curNode
		parFunc.Tails = tailpars
	}
	if *workspace.Timeout {
		return
	}
	parFunc.Pars = &pars
	if (*workspace.Vars)[`_full`] == `1` {
		out = curFunc.Full(parFunc)
	} else {
		out = curFunc.Func(parFunc)
	}
	for key, v := range parFunc.Node.Attr {
		switch attr := v.(type) {
		case string:
			if !strings.HasPrefix(key, `#`) {
				parFunc.Node.Attr[key] = macro(attr, workspace.Vars)
			}
		case map[string]interface{}:
			for parkey, parval := range attr {
				switch parmap := parval.(type) {
				case map[string]interface{}:
					for textkey, textval := range parmap {
						var result interface{}
						switch val := textval.(type) {
						case string:
							result = macro(val, workspace.Vars)
						case []string:
							for i, ival := range val {
								val[i] = macro(ival, workspace.Vars)
							}
							result = val
						}
						if result != nil {
							parFunc.Node.Attr[key].(map[string]interface{})[parkey].(map[string]interface{})[textkey] = result
						}
					}
				}
			}
		}
	}
	for key, v := range parFunc.Node.Attr {
		switch attr := v.(type) {
		case string:
			if strings.HasPrefix(key, `#`) {
				parFunc.Node.Attr[key[1:]] = attr
				delete(parFunc.Node.Attr, key)
			}
		}
	}
	parFunc.Node.Text = macro(parFunc.Node.Text, workspace.Vars)
	for inode, node := range parFunc.Node.Children {
		parFunc.Node.Children[inode].Text = macro(node.Text, workspace.Vars)
	}
	if len(out) > 0 {
		if len(owner.Children) > 0 && owner.Children[len(owner.Children)-1].Tag == tagText {
			owner.Children[len(owner.Children)-1].Text += out
		} else {
			appendText(owner, out)
		}
	}
}

func getFunc(input string, curFunc tplFunc) (*[][]rune, int, *[]*[][]rune) {
	var (
		curp, skip, off, mode, lenParams int
		quote                            bool
		pair, ch                         rune
		tailpar                          *[]*[][]rune
	)
	var params [][]rune
	sizeParam := 32 + len(input)/2
	params = append(params, make([]rune, 0, sizeParam))
	if curFunc.Params == `*` {
		lenParams = 0xff
	} else {
		lenParams = len(strings.Split(curFunc.Params, `,`))
	}
	objLevel := 0
	objMode := 0
	level := 1
	if input[0] == '{' {
		mode = 1
	}
	skip = 1
main:
	for off, ch = range input {
		if skip > 0 {
			skip--
			continue
		}
		if objLevel > 0 {
			params[curp] = append(params[curp], ch)
			switch ch {
			case modes[objMode][0]:
				objLevel++
			case modes[objMode][1]:
				objLevel--
			}
			continue
		}
		if pair > 0 {
			if ch != pair {
				params[curp] = append(params[curp], ch)
			} else {
				if off+1 == len(input) || rune(input[off+1]) != pair {
					pair = 0
					if quote {
						params[curp] = append(params[curp], ch)
						quote = false
					}
				} else {
					params[curp] = append(params[curp], ch)
					skip = 1
				}
			}
			continue
		}
		if len(params[curp]) == 0 && mode == 0 && ch != modes[mode][1] && ch != ',' {
			if ch >= '!' {
				if ch == '"' || ch == '`' {
					pair = ch
				} else if ch == '[' || ch == '{' {
					objMode = 2
					if ch == '{' {
						objMode = 1
					}
					objLevel = 1
					params[curp] = append(params[curp], ch)
				} else {
					if ch == modes[mode][0] {
						level++
					}
					params[curp] = append(params[curp], ch)
				}
			}
			continue
		}

		switch ch {
		case '"', '`':
			if mode == 0 {
				pair = ch
				quote = true
			}
		case ',':
			if mode == 0 && level == 1 && len(params) < lenParams {
				params = append(params, make([]rune, 0, sizeParam))
				curp++
				continue
			}
		case modes[mode][0]:
			level++
		case modes[mode][1]:
			if level > 0 {
				level--
			}
			if level == 0 {
				if mode == 0 && (strings.Contains(curFunc.Params, `Body`) || strings.Contains(curFunc.Params, `Data`)) {
					var isBody bool
					next := off + 1
					for next < len(input) {
						if rune(input[next]) == modes[1][0] {
							isBody = true
							break
						}
						if rune(input[next]) == ' ' || rune(input[next]) == '\t' {
							next++
							continue
						}
						break
					}
					if isBody {
						mode = 1
						for _, keyp := range []string{`Body`, `Data`} {
							if strings.Contains(curFunc.Params, keyp) {
								irune := make([]rune, 0, sizeParam)
								s := keyp + `:`
								params = append(params, append(irune, []rune(s)...))
								break
							}
						}
						curp++
						skip = next - off
						level = 1
						continue
					}
				}
				for tail, ok := tails[curFunc.Tag]; ok && off+2 < len(input) && input[off+1] == '.'; {
					var found bool
					for key, tailFunc := range tail.Tails {
						next := off + 2
						if next < len(input) && strings.HasPrefix(input[next:], key) {
							var isTail bool
							next += len(key)
							for next < len(input) {
								if rune(input[next]) == '(' || rune(input[next]) == '{' {
									isTail = true
									break
								}
								if rune(input[next]) == ' ' || rune(input[next]) == '\t' {
									next++
									continue
								}
								break
							}
							if isTail {
								parTail, shift, _ := getFunc(input[next:], tailFunc.tplFunc)
								off = next
								for ; shift > 0; shift-- {
									_, size := utf8.DecodeRuneInString(input[off:])
									off += size
								}
								if tailpar == nil {
									fortail := make([]*[][]rune, 0)
									tailpar = &fortail
								}
								*parTail = append(*parTail, []rune(key))
								*tailpar = append(*tailpar, parTail)
								found = true
								if tailFunc.Last {
									break main
								}
								break
							}
						}
					}
					if !found {
						break
					}
				}
				break main
			}
		}
		params[curp] = append(params[curp], ch)
		continue
	}
	return &params, utf8.RuneCountInString(input[:off]), tailpar
}

func process(input string, owner *node, workspace *Workspace) {
	var (
		nameOff, shift int
		curFunc        tplFunc
		isFunc         bool
		params         *[][]rune
		tailpars       *[]*[][]rune
	)
	name := make([]rune, 0, 128)
	for off, ch := range input {
		if shift > 0 {
			shift--
			continue
		}
		if ch == '(' {
			if curFunc, isFunc = funcs[string(name[nameOff:])]; isFunc {
				if *workspace.Timeout {
					return
				}
				appendText(owner, macro(string(name[:nameOff]), workspace.Vars))
				name = name[:0]
				nameOff = 0
				params, shift, tailpars = getFunc(input[off:], curFunc)
				callFunc(&curFunc, owner, workspace, params, tailpars)
				for off+shift+3 < len(input) && input[off+shift+1:off+shift+3] == `.(` {
					var next int
					params, next, tailpars = getFunc(input[off+shift+2:], curFunc)
					callFunc(&curFunc, owner, workspace, params, tailpars)
					shift += next + 2
				}
				continue
			}
		}
		if (ch < 'A' || ch > 'Z') && (ch < 'a' || ch > 'z') {
			nameOff = len(name) + 1
		}
		name = append(name, ch)
	}
	appendText(owner, string(name))
}

func parseArg(arg string, workspace *Workspace) (val string) {
	if strings.IndexByte(arg, '(') == -1 {
		return arg
	}

	var owner node
	process(arg, &owner, workspace)
	for _, inode := range owner.Children {
		if inode.Tag == tagText {
			val += inode.Text
		}
	}
	return
}

// Template2JSON converts templates to JSON data
func Template2JSON(input string, timeout *bool, vars *map[string]string) []byte {
	root := node{}
	isvde := (*vars)[`vde`] == `true` || (*vars)[`vde`] == `1`
	sc := smart.SmartContract{
		VDE: isvde,
		VM:  smart.GetVM(),
		TxSmart: tx.SmartContract{
			Header: tx.Header{
				EcosystemID: converter.StrToInt64((*vars)[`ecosystem_id`]),
				KeyID:       converter.StrToInt64((*vars)[`key_id`]),
				NetworkID:   consts.NETWORK_ID,
			},
		},
	}
	process(input, &root, &Workspace{Vars: vars, Timeout: timeout, SmartContract: &sc})
	if root.Children == nil || *timeout {
		return []byte(`[]`)
	}
	for i, v := range root.Children {
		if v.Tag == `text` {
			root.Children[i].Text = macro(v.Text, vars)
		}
	}
	out, err := json.Marshal(root.Children)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.JSONMarshallError, "error": err}).Error("marshalling template data to json")
		return []byte(err.Error())
	}
	return out
}
