#!/usr/bin/env bash
# Seed 5 example devices into a local OpenSOHO instance via the registration endpoint.
# Usage: OPENSOHO_SHARED_SECRET=bad_example ./scripts/seed_devices.sh [http://localhost:8090]

set -euo pipefail

BASE_URL="${1:-http://localhost:8090}"
SECRET="${OPENSOHO_SHARED_SECRET:-bad_example}"

# Each device: key (32 hex chars = 16 bytes), name, mac_address, model, os, hardware_id
declare -a DEVICES=(
  "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4|living-room-ap|a1:b2:c3:d4:e5:f6|GL-MT3000 (Beryl AX)|OpenWrt 24.10.1 r28427|GL-MT3000-001"
  "b2c3d4e5f6a7b2c3d4e5f6a7b2c3d4e5|kitchen-ap|b2:c3:d4:e5:f6:a7|GL-AXT1800 (Slate AX)|OpenWrt 24.10.1 r28427|GL-AXT1800-001"
  "c3d4e5f6a7b8c3d4e5f6a7b8c3d4e5f6|bedroom-ap|c3:d4:e5:f6:a7:b8|TP-Link Archer C7 v5|OpenWrt 24.10.0 r27904|ARCHER-C7-001"
  "d4e5f6a7b8c9d4e5f6a7b8c9d4e5f6a7|garage-ap|d4:e5:f6:a7:b8:c9|Ubiquiti UniFi AP AC Lite|OpenWrt 24.10.0 r27904|UNIFI-APAL-001"
  "e5f6a7b8c9d0e5f6a7b8c9d0e5f6a7b8|office-ap|e5:f6:a7:b8:c9:d0|Netgear WAX206|OpenWrt 24.10.1 r28427|WAX206-001"
)

echo "Registering 5 example devices at ${BASE_URL} ..."
echo

for entry in "${DEVICES[@]}"; do
  IFS='|' read -r key name mac model os hardware_id <<< "${entry}"

  response=$(curl -s -X POST "${BASE_URL}/controller/register/" \
    --data-urlencode "backend=netjsonconfig.OpenWrt" \
    --data-urlencode "key=${key}" \
    --data-urlencode "secret=${SECRET}" \
    --data-urlencode "name=${name}" \
    --data-urlencode "hardware_id=${hardware_id}" \
    --data-urlencode "mac_address=${mac}" \
    --data-urlencode "model=${model}" \
    --data-urlencode "os=${os}" \
    --data-urlencode "system=ath79")

  if echo "${response}" | grep -q "registration-result: success"; then
    uuid=$(echo "${response}" | grep "^uuid:" | awk '{print $2}')
    echo "  OK  ${name}  (uuid: ${uuid})"
  else
    echo "  FAIL  ${name}: ${response}"
  fi
done

echo
echo "Done. Open ${BASE_URL}/_/ to see the devices."
