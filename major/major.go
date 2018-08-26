package major

import (
	"go/printer"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/marwan-at-work/vgop/modfile"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

// Run upgrades or downgrades a module path and
// all of its dependencies.
func Run() {
	op := getOperation()
	modFile := getModFile()
	modName := modFile.Module.Mod.Path
	var newModPath string
	switch op {
	case "upgrade":
		newModPath = getNext(modName)
	case "downgrade":
		newModPath = getPrevious(modName)
	}

	c := &packages.Config{Mode: packages.LoadSyntax}
	pkgs, err := packages.Load(c, "./...")
	must(err)

	for _, p := range pkgs {
		updateImportPath(p, modName, newModPath)
		for _, syn := range p.Syntax {
			goFileName := p.Fset.File(syn.Pos()).Name()
			f, err := os.Create(goFileName)
			must(err)
			defer f.Close()
			must(printer.Fprint(f, p.Fset, syn))
		}
	}
	modFile.Module.Syntax.Token[1] = newModPath
	bts, err := modFile.Format()
	must(err)
	ioutil.WriteFile("go.mod", bts, 0660)
}

func getOperation() string {
	if len(os.Args) != 2 {
		log.Fatal("Use: mod upgrade|downgrade")
	}

	op := os.Args[1]
	if op != "upgrade" && op != "downgrade" {
		log.Fatal("unknown command " + op)
	}

	return op
}

func getNext(s string) string {
	ss := strings.Split(s, "/")
	num, isMajor := versionSuffix(ss)
	if !isMajor {
		return s + "/v2"
	}

	newV := num + 1
	return strings.Join(ss[:len(ss)-1], "/") + "/v" + strconv.Itoa(newV)
}

func versionSuffix(ss []string) (int, bool) {
	last := ss[len(ss)-1]
	if !strings.HasPrefix(last, "v") {
		return 0, false
	}

	numStr := last[1:]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, false
	}

	return num, true
}

func getPrevious(s string) string {
	ss := strings.Split(s, "/")
	num, isMajor := versionSuffix(ss)
	if !isMajor {
		return s
	}

	if num == 2 {
		return strings.Join(ss[:len(ss)-1], "/")
	}

	newV := num - 1
	return strings.Join(ss[:len(ss)-1], "/") + "/v" + strconv.Itoa(newV)
}

func updateImportPath(p *packages.Package, old, new string) {
	for _, syn := range p.Syntax {
		for _, i := range syn.Imports {
			imp := strings.Replace(i.Path.Value, `"`, ``, 2)
			if strings.HasPrefix(imp, old) {
				newImp := strings.Replace(imp, old, new, 1)
				astutil.RewriteImport(p.Fset, syn, imp, newImp)
			}
		}
	}
}

func getModFile() *modfile.File {
	bts, err := ioutil.ReadFile("go.mod")
	must(err)
	dir, err := os.Getwd()
	must(err)
	f, err := modfile.Parse(filepath.Join(dir, "go.mod"), bts, nil)
	must(err)
	return f
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}