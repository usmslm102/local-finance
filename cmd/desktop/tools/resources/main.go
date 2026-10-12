// resources generates Windows icon, version and DPI manifest resources on the
// build host. It is release tooling, never called by the running application.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
)

func main() {
	arch := flag.String("arch", "amd64", "Target architecture")
	buildVersion := flag.String("version", "dev", "Release version")
	flag.Parse()
	if err := generate(*arch, *buildVersion); err != nil {
		log.Fatal(err)
	}
}

func generate(arch, buildVersion string) error {
	architectures := map[string]winres.Arch{"amd64": winres.ArchAMD64, "arm64": winres.ArchARM64}
	target, ok := architectures[arch]
	if !ok {
		return fmt.Errorf("unsupported Windows architecture: %s", arch)
	}
	input, err := os.Open("../../packaging/desktop/AppIcon.png")
	if err != nil {
		return err
	}
	img, err := png.Decode(input)
	input.Close()
	if err != nil {
		return err
	}
	icon, err := winres.NewIconFromImages([]image.Image{img})
	if err != nil {
		return err
	}
	resources := winres.ResourceSet{}
	if err := resources.SetIcon(winres.ID(1), icon); err != nil {
		return err
	}
	resources.SetManifest(winres.AppManifest{
		Description:         "LocalFinance desktop",
		Compatibility:       winres.Win10AndAbove,
		DPIAwareness:        winres.DPIPerMonitorV2,
		ExecutionLevel:      winres.AsInvoker,
		UseCommonControlsV6: true,
		LongPathAware:       true,
	})
	info := version.Info{Type: version.App}
	parts := strings.Split(strings.SplitN(strings.TrimPrefix(buildVersion, "v"), "-", 2)[0], ".")
	for i, part := range parts {
		if i >= 4 {
			break
		}
		n, _ := strconv.ParseUint(part, 10, 16)
		info.FileVersion[i] = uint16(n)
	}
	info.ProductVersion = info.FileVersion
	info.Set(version.LangDefault, version.ProductName, "LocalFinance")
	info.Set(version.LangDefault, version.FileDescription, "LocalFinance desktop — offline personal finance")
	info.Set(version.LangDefault, version.ProductVersion, buildVersion)
	info.Set(version.LangDefault, version.FileVersion, buildVersion)
	resources.SetVersionInfo(info)
	output, err := os.Create("rsrc_windows_" + arch + ".syso")
	if err != nil {
		return err
	}
	if err := resources.WriteObject(output, target); err != nil {
		output.Close()
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	ico, err := os.Create("../../dist/desktop/AppIcon.ico")
	if err != nil {
		return err
	}
	if err := icon.SaveICO(ico); err != nil {
		ico.Close()
		return err
	}
	return ico.Close()
}
