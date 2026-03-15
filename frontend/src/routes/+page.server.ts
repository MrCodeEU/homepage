import { readFileSync } from 'fs';
import { resolve } from 'path';
import type { PageServerLoad } from './$types';
import type { Project, StravaData, LinkedInData } from '$lib/api';

// Reads a generated JSON wrapper and returns the inner `data` field.
// Path is relative to the frontend/ working directory (one level up → repo root).
// Returns null if the file is missing (e.g. during Docker frontend-only build).
function loadJson<T>(filename: string): T | null {
	try {
		const filePath = resolve(process.cwd(), '..', 'backend', 'data', 'generated', filename);
		const raw = readFileSync(filePath, 'utf-8');
		return JSON.parse(raw).data as T;
	} catch {
		return null;
	}
}

export const load: PageServerLoad = () => {
	return {
		projects: loadJson<Project[]>('github.json') ?? [],
		strava: loadJson<StravaData>('strava.json'),
		linkedIn: loadJson<LinkedInData>('linkedin.json')
	};
};
