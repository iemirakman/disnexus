package monitor

import (
	"context"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Monitor sistemdeki aktif ağ adaptörlerini izler ve
// TargetCIDR bloğuna uyan IPv4 adreslerini filtreler.
type Monitor struct {
	cidr      *net.IPNet
	activeIPs []net.IP
	mu        sync.RWMutex
	rrIndex   uint64
	onChange  chan []net.IP
	cancel    context.CancelFunc
}

// NewMonitor yeni bir Monitor oluşturur.
func NewMonitor(cidr string) (*Monitor, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	return &Monitor{
		cidr:     ipnet,
		onChange: make(chan []net.IP, 5),
	}, nil
}

// OnChange aktif IP listesi değiştiğinde sinyal alan channel'ı döner.
func (m *Monitor) OnChange() <-chan []net.IP {
	return m.onChange
}

// Start ağ taramasını 3 saniye aralıklarla başlatır.
func (m *Monitor) Start(ctx context.Context) {
	subCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	m.scan() // İlk tarama hemen yapılsın

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.scan()
			case <-subCtx.Done():
				return
			}
		}
	}()
}

// Stop taramayı ve arka plan goroutine'ini durdurur.
func (m *Monitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
}

// GetActiveIPs thread-safe şekilde aktif IP listesini döner.
func (m *Monitor) GetActiveIPs() []net.IP {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ips := make([]net.IP, len(m.activeIPs))
	copy(ips, m.activeIPs)
	return ips
}

// GetNextIP Round-Robin algoritması ile sıradaki aktif IP'yi seçer.
func (m *Monitor) GetNextIP() (net.IP, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.activeIPs) == 0 {
		return nil, false
	}
	idx := atomic.AddUint64(&m.rrIndex, 1)
	ip := m.activeIPs[int(idx)%len(m.activeIPs)]
	return ip, true
}

// scan sistemdeki ağ arayüzlerini tarar ve aktif IP listesini günceller (Debounce / State-Check).
func (m *Monitor) scan() {
	var found []net.IP

	ifaces, err := net.Interfaces()
	if err != nil {
		log.Printf("[monitor] arayüzler alınamadı: %v", err)
		return
	}

	for _, i := range ifaces {
		// Yalnızca up ve loopback olmayan arayüzler
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				ip := ipnet.IP.To4()
				if ip != nil && m.cidr.Contains(ip) {
					found = append(found, ip)
				}
			}
		}
	}

	// Değişiklik kontrolü (State-Check / Debounce)
	if m.hasChanged(found) {
		m.mu.Lock()
		m.activeIPs = found
		m.mu.Unlock()

		log.Printf("[monitor] Aktif IP listesi güncellendi: %d adet aktif hat", len(found))

		// Non-blocking channel notify
		select {
		case m.onChange <- found:
		default:
		}
	}
}

// hasChanged iki IP listesinin aynı olup olmadığını kontrol eder.
func (m *Monitor) hasChanged(newIPs []net.IP) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.activeIPs) != len(newIPs) {
		return true
	}

	for i := range m.activeIPs {
		if !m.activeIPs[i].Equal(newIPs[i]) {
			return true
		}
	}
	return false
}
