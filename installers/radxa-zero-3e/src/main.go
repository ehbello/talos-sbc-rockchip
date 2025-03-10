// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/siderolabs/go-cmd/pkg/cmd"
	"github.com/siderolabs/go-copy/copy"
	"github.com/siderolabs/talos/pkg/machinery/overlay"
	"github.com/siderolabs/talos/pkg/machinery/overlay/adapter"
	"golang.org/x/sys/unix"
)

const (
	off   int64 = 512 * 64
	board       = "radxa-zero-3e"
	dtb         = "rockchip/rk3566-radxa-zero-3e.dtb"
)

func main() {
	adapter.Execute(&radxaZero3e{})
}

type radxaZero3e struct{}

type radxaZero3eExtraOptions struct {
	DTOverlays string `yaml:"dtOverlays,omitempty"`
}

func (i *radxaZero3e) GetOptions(extra radxaZero3eExtraOptions) (overlay.Options, error) {
	return overlay.Options{
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
	}, nil
}

func (i *radxaZero3e) Install(options overlay.InstallOptions[radxaZero3eExtraOptions]) error {
	uBootBin := filepath.Join(options.ArtifactsPath, "arm64/u-boot", board, "u-boot-rockchip.bin")

	if err := uBootLoaderInstall(uBootBin, options.InstallDisk); err != nil {
		return err
	}

	src := filepath.Join(options.ArtifactsPath, "arm64/dtb", dtb)
	dst := filepath.Join(options.MountPrefix, "boot/EFI/dtb", dtb)

	if dtOverlays := options.ExtraOptions.DTOverlays; dtOverlays != "" {
		// Apply each overlay sequentially
		overlayNames := strings.Split(dtOverlays, ",")
		fdtoverlayPath := filepath.Join(options.ArtifactsPath, "arm64/fdtoverlay")

		for _, overlayName := range overlayNames {
			overlayPath := filepath.Join(options.ArtifactsPath, "arm64/dtb/rockchip/overlays", overlayName+".dtbo")

			// Run fdtoverlay to merge the overlay with the base DTB
			if _, err := cmd.Run(
				fdtoverlayPath,
				"-v",      // verbose output
				"-i", src, // input file
				"-o", src, // output file (it is ok to use the same file)
				overlayPath, // overlay file
			); err != nil {
				return fmt.Errorf("failed to apply overlay %s: %w", overlayName, err)
			}
		}
	}

	return copyFileAndCreateDir(src, dst)
}

func copyFileAndCreateDir(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}

	return copy.File(src, dst)
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
