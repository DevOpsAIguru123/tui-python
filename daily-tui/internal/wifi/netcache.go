package wifi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

var cacheFilePath = filepath.Join(os.Getenv("HOME"), ".config", "daily-tui", "network-cache.json")

// networkCache maps a network fingerprint key → SSID.
// The key is "<gatewayIP>/<gatewayMAC>" for maximum uniqueness:
//   - gatewayIP  alone can collide (many routers default to 192.168.1.1)
//   - gatewayMAC alone requires an ARP lookup on every read
//   - combining both is unique and the IP is free from ipconfig output
//
// If the MAC is not available (ARP miss), the IP alone is used as key.
type networkCache map[string]string

func loadNetworkCache() networkCache {
	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		return networkCache{}
	}
	var cache networkCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return networkCache{}
	}
	return cache
}

func saveNetworkCache(cache networkCache) {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(cacheFilePath, data, 0644)
}

// networkKey builds the cache lookup key from gateway IP and MAC.
// If mac is empty (ARP unavailable), falls back to IP-only key.
func networkKey(ip, mac string) string {
	if mac != "" {
		return ip + "/" + mac
	}
	return ip
}

// LookupCachedSSID returns the SSID for the given gateway IP+MAC, or "".
// Tries the full IP/MAC key first, then the IP-only key for legacy entries.
func LookupCachedSSID(gatewayIP, gatewayMAC string) string {
	if gatewayIP == "" {
		return ""
	}
	cache := loadNetworkCache()
	if ssid := cache[networkKey(gatewayIP, gatewayMAC)]; ssid != "" {
		return ssid
	}
	// Fallback: IP-only key (covers entries saved without MAC)
	return cache[gatewayIP]
}

// SaveCachedSSID stores the gateway → SSID mapping to disk.
func SaveCachedSSID(gatewayIP, gatewayMAC, ssid string) {
	if gatewayIP == "" || ssid == "" || ssid == "Wi-Fi" {
		return
	}
	cache := loadNetworkCache()
	cache[networkKey(gatewayIP, gatewayMAC)] = ssid
	saveNetworkCache(cache)
}

// ParseARPMAC extracts the MAC from `arp -n <ip>` output.
// Output format: "? (x.x.x.x) at aa:bb:cc:dd:ee:ff on en0 ..."
// Returns "" if the MAC cannot be parsed or is "(incomplete)".
func ParseARPMAC(output string) string {
	idx := strings.Index(output, " at ")
	if idx == -1 {
		return ""
	}
	rest := strings.TrimSpace(output[idx+4:])
	if len(rest) == 0 {
		return ""
	}
	mac := strings.Fields(rest)[0]
	if strings.Count(mac, ":") == 5 {
		return mac
	}
	return ""
}
