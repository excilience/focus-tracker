CREATE UNIQUE INDEX activities_active_title_unique_idx
ON activities (lower(trim(title)))
WHERE is_archived = false;