package domain

type (
	TrivyReportPayload struct {
		ReportName      string                `json:"report_name"`
		Namespace       string                `json:"namespace"`
		Vulnerabilities []VulnerabilityDetail `json:"vulnerabilities"`
		Critical        int                   `json:"critical"`
		High            int                   `json:"high"`
	}

	VulnerabilityDetail struct {
		VulnerabilityID  string `json:"vulnerability_id"`
		PkgName          string `json:"pkg_name"`
		InstalledVersion string `json:"installed_version"`
		FixedVersion     string `json:"fixed_version"`
		Severity         string `json:"severity"`
		Title            string `json:"title"`
	}

	TrivyVulnerability struct {
		ReportName      string                `json:"report_name"`
		Namespace       string                `json:"namespace"`
		WorkloadKind    string                `json:"workload_kind"`
		WorkloadName    string                `json:"workload_name"`
		ContainerName   string                `json:"container_name"`
		Critical        int                   `json:"critical"`
		High            int                   `json:"high"`
		Medium          int                   `json:"medium"`
		Low             int                   `json:"low"`
		Action          string                `json:"action"`
		Vulnerabilities []VulnerabilityDetail `json:"vulnerabilities"`
	}
)
