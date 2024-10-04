echo "Applying base device tree: ${fdtfile}"
load ${devtype} ${devnum}:${distro_bootpart} ${fdt_addr_r} ${prefix}dtb/${fdtfile}
fdt addr ${fdt_addr_r}

echo "Resizing the fdt to 65536"
fdt resize 65536

echo "Configuring fdt_overlay environment variable"
setenv fdt_overlays tpm-slb9670-overlay.dtbo

for overlay in $fdt_overlays; do
    echo "Applying overlay: $overlay"
    load ${devtype} ${devnum}:${distro_bootpart} ${fdtoverlay_addr_r} ${prefix}dtb/overlays/${overlay}
    fdt apply ${fdtoverlay_addr_r}
done

bootefi bootmgr ${fdt_addr_r}
