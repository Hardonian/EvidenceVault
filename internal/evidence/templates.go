package evidence

import "time"

type StarterTemplate struct {
	Key, Name, Description string
	Item                   Item
}

func StarterTemplates(now time.Time) []StarterTemplate {
	expSoon := now.AddDate(0, 1, 0)
	expLong := now.AddDate(1, 0, 0)
	return []StarterTemplate{
		{Key: "insurance-certificate", Name: "Insurance Certificate", Description: "Track business liability or cyber insurance renewal.", Item: Item{Title: "Insurance Certificate", Category: "Compliance", Status: "expiring", ReminderDaysBefore: 30, ExpiryDate: &expSoon, OwnerName: "Finance Owner", OwnerEmail: "finance@example.com", ControlRefs: []string{"Renewal Tracking"}, VendorRefs: []string{"Insurance Carrier"}, RiskRefs: []string{"insurance lapse"}, Notes: "Starter template for insurance evidence."}},
		{Key: "privacy-policy", Name: "Privacy Policy", Description: "Maintain current policy publication and review ownership.", Item: Item{Title: "Privacy Policy", Category: "Legal", Status: "active", ReminderDaysBefore: 30, ExpiryDate: &expLong, OwnerName: "Legal Owner", OwnerEmail: "legal@example.com", ControlRefs: []string{"Privacy Notice"}, RiskRefs: []string{"privacy drift"}, Notes: "Starter template for policy evidence."}},
		{Key: "vendor-agreement", Name: "Vendor Agreement", Description: "Track critical supplier contracts and renewal windows.", Item: Item{Title: "Vendor Agreement", Category: "Legal", Status: "active", ReminderDaysBefore: 45, ExpiryDate: &expLong, OwnerName: "Operations Owner", OwnerEmail: "ops@example.com", ControlRefs: []string{"Vendor Management"}, VendorRefs: []string{"Critical Supplier"}, RiskRefs: []string{"third-party continuity"}, Notes: "Starter template for vendor agreement evidence."}},
		{Key: "soc2-evidence", Name: "SOC2 Evidence", Description: "Store and refresh SOC2 related policy or control artifacts.", Item: Item{Title: "SOC2 Evidence", Category: "Security", Status: "active", ReminderDaysBefore: 30, ExpiryDate: &expSoon, OwnerName: "Security Owner", OwnerEmail: "security@example.com", ControlRefs: []string{"Security Control Evidence"}, RiskRefs: []string{"control evidence drift"}, Notes: "Starter template for SOC2 evidence."}},
		{Key: "accessibility-statement", Name: "Accessibility Statement", Description: "Track statement updates and publication evidence.", Item: Item{Title: "Accessibility Statement", Category: "Compliance", Status: "active", ReminderDaysBefore: 30, ExpiryDate: &expLong, OwnerName: "Product Owner", OwnerEmail: "product@example.com", ControlRefs: []string{"Accessibility Review"}, RiskRefs: []string{"public trust"}, Notes: "Starter template for accessibility statement evidence."}},
		{Key: "employee-handbook-ack", Name: "Employee Handbook Acknowledgement", Description: "Record annual handbook acknowledgment collection.", Item: Item{Title: "Employee Handbook Acknowledgement", Category: "HR", Status: "expiring", ReminderDaysBefore: 30, ExpiryDate: &expSoon, OwnerName: "People Ops", OwnerEmail: "people@example.com", ControlRefs: []string{"HR Acknowledgement"}, RiskRefs: []string{"policy acknowledgement"}, Notes: "Starter template for handbook acknowledgment evidence."}},
	}
}

func TemplateByKey(now time.Time, key string) (StarterTemplate, bool) {
	for _, t := range StarterTemplates(now) {
		if t.Key == key {
			return t, true
		}
	}
	return StarterTemplate{}, false
}
