package email

type MonitorService struct {
	config  IMAPConfig
	monitor Monitor
}

func NewMonitorService(config IMAPConfig, monitor Monitor) *MonitorService {
	return &MonitorService{config: config, monitor: monitor}
}

func (m *MonitorService) Start() error {
	return ErrMonitorUnsupported
}

func (m *MonitorService) Stop() {}
