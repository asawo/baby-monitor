// API response types — mirrors Go server structs

/** @typedef {{ detected_at: string | null, seconds_ago: number }} CryStatus */
/** @typedef {{ error: string | null }} DetectStatus */
/** @typedef {{ name: string, active: boolean }} ServiceStatus */
/** @typedef {{ enabled: boolean }} NotificationsState */
/** @typedef {{ name: string, content: string }} LogSection */

export {};
