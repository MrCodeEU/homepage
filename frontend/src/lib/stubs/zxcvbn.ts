// Stub for zxcvbn-typescript — the real library is 809KB and only used by
// mljr-svelte's Password component which this site doesn't use.
// This no-op keeps the Password component from crashing if somehow invoked.
export default function zxcvbn() {
	return {
		score: 0,
		feedback: { warning: '', suggestions: [] },
		crack_times_display: {
			online_throttling_100_per_hour: '',
			online_no_throttling_10_per_second: '',
			offline_slow_hashing_1e4_per_second: '',
			offline_fast_hashing_1e10_per_second: ''
		}
	};
}
