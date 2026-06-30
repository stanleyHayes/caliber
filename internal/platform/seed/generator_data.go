package seed

import "github.com/xcreativs/caliber/internal/domain/role"

// familyTemplate describes a candidate role family used to generate realistic,
// locally-plausible CVs. The skills are deliberately chosen so that the real
// parser can extract them with CV evidence (no fabrication).
type familyTemplate struct {
	name         string
	degree       string
	targetTitles []string
	skills       []string
	tools        []string
}

// roleTemplate describes a generated open role. The free-text prompt built from
// it is fed to the real role-spec parser.
type roleTemplate struct {
	employerIdx      int
	title            string
	location         string
	seniority        role.Seniority
	availability     string
	responsibilities []string
	mustHaves        []string
	niceToHaves      []string
	compensation     string
}

// generatorTemplates returns the deterministic tables that drive the generation
// pipeline. Keeping them in a function avoids package-level mutable state and
// keeps the generator reproducible without any global random source.
//
//nolint:funlen // a flat, readable seed-data table; not logic.
func generatorTemplates() struct {
	firstNames   []string
	lastNames    []string
	locations    []string
	universities []string
	companies    []string
	families     []familyTemplate
	roles        []roleTemplate
} {
	return struct {
		firstNames   []string
		lastNames    []string
		locations    []string
		universities []string
		companies    []string
		families     []familyTemplate
		roles        []roleTemplate
	}{
		firstNames: []string{
			"Ama", "Kofi", "Esi", "Yaw", "Abena", "Kwame", "Adwoa", "Kojo",
			"Akua", "Kwesi", "Afia", "Kwabena", "Efua", "Yaa", "Kwaku",
		},
		lastNames: []string{
			"Mensah", "Asante", "Owusu", "Boateng", "Sarpong", "Boadu", "Agyeman",
			"Antwi", "Addo", "Osei", "Adu", "Tetteh", "Ankrah", "Darko", "Appiah",
		},
		locations: []string{
			locAccra, locKumasi, "Takoradi", "Tamale", "Cape Coast", locRemote,
		},
		universities: []string{
			"Ashesi University",
			"Kwame Nkrumah University of Science and Technology",
			"University of Ghana",
			"University of Cape Coast",
			"Academic City College",
			"Lancaster University Ghana",
		},
		companies: []string{
			"Hubtel", "mPharma", "MTN Ghana", "Andela Ghana", "Fidelity Bank Ghana",
			"Tullow Oil Ghana", "Standard Chartered Bank Ghana", "GCB Bank",
			"Ecobank Ghana", "Paystack", "Flutterwave Ghana", "Wave Ghana",
			"Safaricom", "BCX Ghana", "Vodafone Ghana",
		},
		families: []familyTemplate{
			{
				name:         "Backend Engineering",
				degree:       "BSc Computer Science",
				targetTitles: []string{"Backend Engineer", "Software Engineer"},
				skills:       []string{"Go", "Python", "SQL", "Postgres", "gRPC", "System design"},
				tools:        []string{"Docker", "Kubernetes", "AWS"},
			},
			{
				name:         "Frontend Engineering",
				degree:       "BSc Computer Science",
				targetTitles: []string{"Frontend Engineer", "Software Engineer"},
				skills:       []string{"TypeScript", "React", "JavaScript", "HTML/CSS"},
				tools:        []string{"Docker", "AWS"},
			},
			{
				name:         "Mobile Engineering",
				degree:       "BSc Computer Science",
				targetTitles: []string{"Mobile Engineer", "Software Engineer"},
				skills:       []string{"TypeScript", "React", "JavaScript", "Communication"},
				tools:        []string{"Docker", "AWS"},
			},
			{
				name:         "Data Engineering",
				degree:       "BSc Computer Engineering",
				targetTitles: []string{"Data Engineer", "Software Engineer"},
				skills:       []string{"Python", "SQL", "Postgres", "Communication"},
				tools:        []string{"Airflow", "dbt", "AWS"},
			},
			{
				name:         "Platform Engineering",
				degree:       "BSc Computer Engineering",
				targetTitles: []string{"Platform Engineer", "Software Engineer"},
				skills:       []string{"Go", "Kubernetes", "Docker", "AWS"},
				tools:        []string{"Terraform", "CI/CD", "gRPC"},
			},
			{
				name:         "DevOps Engineering",
				degree:       "BSc Computer Engineering",
				targetTitles: []string{"DevOps Engineer", "Software Engineer"},
				skills:       []string{"Kubernetes", "Docker", "AWS", "Communication"},
				tools:        []string{"Terraform", "CI/CD"},
			},
			{
				name:         "Machine Learning Engineering",
				degree:       "BSc Computer Science",
				targetTitles: []string{"Machine Learning Engineer", "Software Engineer"},
				skills:       []string{"Python", "SQL", "Communication"},
				tools:        []string{"TensorFlow", "AWS"},
			},
			{
				name:         "Full-Stack Engineering",
				degree:       "BSc Computer Science",
				targetTitles: []string{"Full-Stack Engineer", "Software Engineer"},
				skills:       []string{"TypeScript", "React", "Python", "SQL", "Postgres"},
				tools:        []string{"Docker", "AWS", "gRPC"},
			},
			{
				name:         "QA Engineering",
				degree:       "BSc Computer Science",
				targetTitles: []string{"QA Engineer", "Software Engineer"},
				skills:       []string{"Communication", "JavaScript"},
				tools:        []string{"Selenium", "Cypress"},
			},
			{
				name:         "Product Management",
				degree:       "BSc Business Administration",
				targetTitles: []string{"Product Manager", "Product Owner"},
				skills:       []string{"Communication", "Data analysis"},
				tools:        []string{"Jira", "SQL"},
			},
		},
		roles: []roleTemplate{
			{
				employerIdx: 0, title: "Senior Backend Engineer", location: locAccra,
				seniority: role.SenioritySenior, availability: "within 1 month",
				responsibilities: []string{
					"Design and operate high-throughput backend services",
					"Mentor junior engineers and review critical code",
				},
				mustHaves:    []string{"Go", "SQL", "System design"},
				niceToHaves:  []string{"Kubernetes", "AWS"},
				compensation: "GHS 12,000 - 20,000 per month",
			},
			{
				employerIdx: 1, title: "Data Engineer", location: locRemote,
				seniority: role.SeniorityMid, availability: "within 2 months",
				responsibilities: []string{
					"Build reliable data pipelines and warehouses",
					"Partner with analysts to surface trusted metrics",
				},
				mustHaves:    []string{"Python", "SQL"},
				niceToHaves:  []string{"Airflow", "dbt"},
				compensation: "GHS 9,000 - 16,000 per month",
			},
			{
				employerIdx: 2, title: "Mobile Engineer", location: locAccra,
				seniority: role.SeniorityMid, availability: "within 1 month",
				responsibilities: []string{
					"Ship cross-platform mobile features used across West Africa",
					"Collaborate with product and design on user experience",
				},
				mustHaves:    []string{"TypeScript", "React"},
				niceToHaves:  []string{"React Native", "Communication"},
				compensation: "GHS 8,000 - 14,000 per month",
			},
			{
				employerIdx: 0, title: "Platform Engineer", location: locAccra,
				seniority: role.SenioritySenior, availability: "within 3 months",
				responsibilities: []string{
					"Own cloud infrastructure and internal developer platforms",
					"Drive reliability engineering and incident response",
				},
				mustHaves:    []string{"Go", "Kubernetes"},
				niceToHaves:  []string{"AWS", "Terraform"},
				compensation: "GHS 13,000 - 22,000 per month",
			},
			{
				employerIdx: 1, title: "Junior Frontend Engineer", location: locKumasi,
				seniority: role.SeniorityJunior, availability: "immediately",
				responsibilities: []string{
					"Implement accessible UI components under senior guidance",
					"Write tests and document reusable frontend patterns",
				},
				mustHaves:    []string{"React", "TypeScript"},
				niceToHaves:  []string{"HTML/CSS", "Communication"},
				compensation: "GHS 4,000 - 7,000 per month",
			},
			{
				employerIdx: 3, title: "Product Manager", location: locAccra,
				seniority: role.SenioritySenior, availability: "within 1 month",
				responsibilities: []string{
					"Define roadmap and prioritise engineering investments",
					"Run discovery with Ghana-based enterprise customers",
				},
				mustHaves:    []string{"Communication", "Data analysis"},
				niceToHaves:  []string{"SQL", "Jira"},
				compensation: "GHS 12,000 - 20,000 per month",
			},
			{
				employerIdx: 4, title: "QA Engineer", location: locRemote,
				seniority: role.SeniorityMid, availability: "within 1 month",
				responsibilities: []string{
					"Design automated test suites for mobile and web releases",
					"Report and track defects through to resolution",
				},
				mustHaves:    []string{"Selenium", "Communication"},
				niceToHaves:  []string{"Cypress", "JavaScript"},
				compensation: "GHS 7,000 - 12,000 per month",
			},
			{
				employerIdx: 5, title: "DevOps Engineer", location: locAccra,
				seniority: role.SeniorityMid, availability: "within 2 months",
				responsibilities: []string{
					"Maintain CI/CD pipelines and cloud deployments",
					"Improve observability and infrastructure security",
				},
				mustHaves:    []string{"Kubernetes", "Docker"},
				niceToHaves:  []string{"AWS", "Terraform"},
				compensation: "GHS 9,000 - 15,000 per month",
			},
			{
				employerIdx: 3, title: "Machine Learning Engineer", location: locAccra,
				seniority: role.SenioritySenior, availability: "within 3 months",
				responsibilities: []string{
					"Train and deploy production ML models for fraud detection",
					"Build feature pipelines and monitor model drift",
				},
				mustHaves:    []string{"Python", "SQL"},
				niceToHaves:  []string{"TensorFlow", "AWS"},
				compensation: "GHS 14,000 - 23,000 per month",
			},
			{
				employerIdx: 6, title: "Full-Stack Engineer", location: locRemote,
				seniority: role.SeniorityMid, availability: "within 1 month",
				responsibilities: []string{
					"Build end-to-end features across web and API layers",
					"Support cross-functional squads with rapid prototyping",
				},
				mustHaves:    []string{"TypeScript", "React", "SQL"},
				niceToHaves:  []string{"Python", "AWS"},
				compensation: "GHS 9,000 - 16,000 per month",
			},
			{
				employerIdx: 7, title: "Site Reliability Engineer", location: locAccra,
				seniority: role.SenioritySenior, availability: "within 2 months",
				responsibilities: []string{
					"Ensure 99.9% uptime for payments infrastructure",
					"Run blameless post-mortems and build reliability tooling",
				},
				mustHaves:    []string{"Kubernetes", "Go"},
				niceToHaves:  []string{"AWS", "gRPC"},
				compensation: "GHS 14,000 - 23,000 per month",
			},
			{
				employerIdx: 4, title: "Security Engineer", location: locAccra,
				seniority: role.SenioritySenior, availability: "within 1 month",
				responsibilities: []string{
					"Secure cloud workloads and enforce compliance controls",
					"Lead vulnerability management and threat modelling",
				},
				mustHaves:    []string{"Python", "AWS"},
				niceToHaves:  []string{"Kubernetes", "Communication"},
				compensation: "GHS 13,000 - 22,000 per month",
			},
		},
	}
}
