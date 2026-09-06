ALTER TABLE templates DROP CONSTRAINT IF EXISTS uk_templates_proj_type_chan;
ALTER TABLE templates DROP COLUMN IF EXISTS name;
ALTER TABLE templates DROP COLUMN IF EXISTS created_at;
