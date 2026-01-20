const API_BASE = import.meta.env.DEV ? 'http://localhost:8080' : '';

// LinkedIn / CV Data
export interface LinkedInData {
	profile: LinkedInProfile;
	experience: LinkedInExperience[];
	education: LinkedInEducation[];
	skills: string[];
}

export interface LinkedInProfile {
	name: string;
	headline: string;
	location: string;
	summary: string;
	photo_url?: string;
}

export interface LinkedInExperience {
	title: string;
	company: string;
	company_logo?: string;
	location: string;
	start_date: string;
	end_date: string;
	description: string;
	duration?: string;
}

export interface LinkedInEducation {
	school: string;
	school_logo?: string;
	degree: string;
	field: string;
	start_date: string;
	end_date: string;
	description?: string;
}

// Legacy CV interface (for compatibility)
export interface CV {
	name: string;
	title: string;
	summary: string;
	experience: Experience[];
	education: Education[];
	skills: string[];
}

export interface Experience {
	title: string;
	company: string;
	location: string;
	start_date: string;
	end_date: string;
	description: string;
}

export interface Education {
	school: string;
	degree: string;
	field: string;
	start_date: string;
	end_date: string;
}

// GitHub Projects
export interface ProjectLink {
	name: string;
	url: string;
	icon?: string; // Optional Material Design icon name (e.g., "mdi:rocket-launch")
}

export interface Project {
	name: string;
	description: string;
	url: string;
	stars: number;
	language: string;
	topics: string[];
	images: string[];
	featured: boolean;
	links: ProjectLink[];
}

// Strava Data
export interface StravaData {
	total_stats: StravaStats;
	year_to_date_stats: StravaStats;
	recent_activities: StravaActivity[];
	best_activities: StravaBestRecords;
	personal_records: StravaRecord[];
}

export interface StravaStats {
	count: number;
	distance: number;
	moving_time: number;
	elapsed_time: number;
	elevation_gain: number;
}

export interface StravaActivity {
	id: number;
	name: string;
	distance: number;
	moving_time: number;
	elapsed_time: number;
	total_elevation_gain: number;
	type: string;
	start_date: string;
	average_pace: number;
	average_speed: number;
	max_speed: number;
	average_heartrate?: number;
	max_heartrate?: number;
}

export interface StravaBestRecords {
	longest_distance: StravaActivity;
	longest_time: StravaActivity;
	fastest_pace: StravaActivity;
	most_elevation: StravaActivity;
}

export interface StravaRecord {
	type: string;
	time: number;
	distance: number;
	date: string;
	activity: StravaActivity;
}

// Legacy interface (for compatibility)
export interface Activity {
	name: string;
	distance: number;
	moving_time: number;
	date: string;
}

// New API functions with updated types
export async function getLinkedInData(): Promise<LinkedInData> {
	const res = await fetch(`${API_BASE}/api/cv`);
	if (!res.ok) throw new Error('Failed to fetch LinkedIn data');
	return res.json();
}

export async function getProjects(): Promise<Project[]> {
	const res = await fetch(`${API_BASE}/api/projects`);
	if (!res.ok) throw new Error('Failed to fetch projects');
	return res.json();
}

export async function getStravaData(): Promise<StravaData> {
	const res = await fetch(`${API_BASE}/api/strava`);
	if (!res.ok) throw new Error('Failed to fetch Strava data');
	return res.json();
}

// Legacy functions for compatibility
export async function getCV(): Promise<CV> {
	const linkedInData = await getLinkedInData();
	// Convert LinkedInData to legacy CV format
	return {
		name: linkedInData.profile.name,
		title: linkedInData.profile.headline,
		summary: linkedInData.profile.summary,
		experience: linkedInData.experience,
		education: linkedInData.education,
		skills: linkedInData.skills
	};
}

export async function getStravaStats(): Promise<Activity[]> {
	const stravaData = await getStravaData();
	// Convert to legacy format
	return stravaData.recent_activities.map(activity => ({
		name: activity.name,
		distance: activity.distance,
		moving_time: activity.moving_time,
		date: activity.start_date
	}));
}
