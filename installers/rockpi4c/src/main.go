// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/siderolabs/talos/pkg/machinery/overlay"
	"github.com/siderolabs/talos/pkg/machinery/overlay/adapter"
	"golang.org/x/sys/unix"
)

const (
	off   int64 = 512 * 64
	board       = "rockpi4c"

	// Artifacts-relative device tree paths (see installers/pkg.yaml, which
	// bundles the base DTB, the radxa-overlays .dtbo files and a static
	// fdtoverlay into the overlay image).
	baseDTB        = "arm64/dtb/rockchip/rk3399-rock-pi-4c.dtb"
	overlaysDir    = "arm64/dtb/rockchip/overlays"
	fdtoverlayTool = "arm64/fdtoverlay"
)

func main() {
	adapter.Execute(context.Background(), &rockPi4c{})
}

type rockPi4c struct{}

type rockPi4cExtraOptions struct {
	DTOverlays string `yaml:"dtOverlays,omitempty"`
}

func (i *rockPi4c) GetOptions(_ context.Context, extra rockPi4cExtraOptions) (overlay.Options, error) {
	options := overlay.Options{
		Name: board,
		KernelArgs: []string{
			"console=tty0",
			"console=ttyS2,1500000n8",
			"sysctl.kernel.kexec_load_disabled=1",
			"talos.dashboard.disabled=1",
		},
		PartitionOptions: overlay.PartitionOptions{
			Offset: 2048 * 10,
		},
		// Embed the (measured) base device tree in the UKI. Board overlays are
		// opt-in via the dtOverlays extra option and merged on top at image
		// build time, so they end up inside the signed and measured UKI.
		DeviceTree: baseDTB,
	}

	if dtOverlays := deviceTreeOverlays(extra.DTOverlays); len(dtOverlays) > 0 {
		options.DeviceTreeOverlays = dtOverlays
		options.DeviceTreeOverlayTool = fdtoverlayTool
	}

	return options, nil
}

// deviceTreeOverlays maps a comma-separated list of overlay names (radxa-overlays
// .dtbo basenames, e.g. "rk3399-spi1-cs-gpio-slb9670") to their artifacts-relative
// .dtbo paths.
func deviceTreeOverlays(dtOverlays string) []string {
	var overlays []string

	for _, name := range strings.Split(dtOverlays, ",") {
		if name = strings.TrimSpace(name); name != "" {
			overlays = append(overlays, filepath.Join(overlaysDir, name+".dtbo"))
		}
	}

	return overlays
}

func (i *rockPi4c) Install(_ context.Context, options overlay.InstallOptions[rockPi4cExtraOptions]) error {
	uBootBin := filepath.Join(options.ArtifactsPath, "arm64/u-boot", board, "u-boot-rockchip.bin")

	return uBootLoaderInstall(uBootBin, options.InstallDisk)
}

func uBootLoaderInstall(uBootBin, installDisk string) error {
	f, err := os.OpenFile(installDisk, unix.O_RDWR|unix.O_CLOEXEC, 0o666)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", installDisk, err)
	}

	defer f.Close() //nolint:errcheck

	uboot, err := os.ReadFile(uBootBin)
	if err != nil {
		return err
	}

	if _, err = f.WriteAt(uboot, off); err != nil {
		return err
	}

	// NB: In the case that the block device is a loopback device, we sync here
	// to ensure that the file is written before the loopback device is
	// unmounted.
	return f.Sync()
}
