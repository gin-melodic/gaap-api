-- Themes
INSERT INTO themes (id, name, is_dark, colors) VALUES
-- 1. Classic (Default)
('01938d64-5c6b-7d24-8f3a-000000000001'::uuid, 'Classic Blue', FALSE, '{"primary": "#4F46E5", "bg": "#F8FAFC", "card": "#FFFFFF", "text": "#1E293B", "muted": "#64748B", "border": "#E2E8F0"}'),
-- 2. Rose
('01938d64-5c6b-7d24-8f3a-000000000002'::uuid, 'Retro Red (Rose)', FALSE, '{"primary": "#D13C58", "bg": "#E3D4B5", "card": "#FDFBF7", "text": "#4A0404", "muted": "#8C6B6B", "border": "#D4C5A9"}'),
-- 3. Midnight
('01938d64-5c6b-7d24-8f3a-000000000003'::uuid, 'Midnight Purple', FALSE, '{"primary": "#3A022B", "bg": "#E3E7F3", "card": "#FFFFFF", "text": "#2D1B36", "muted": "#7A6E85", "border": "#D1D5DB"}'),
-- 4. Dark
('01938d64-5c6b-7d24-8f3a-000000000004'::uuid, 'Night Mode (Dark)', TRUE, '{"primary": "#D4C5B0", "bg": "#2A2A2E", "card": "#38383C", "text": "#E3E3E3", "muted": "#A1A1AA", "border": "#45454A"}'),
-- 5. Pop
('01938d64-5c6b-7d24-8f3a-000000000005'::uuid, 'Pop Style (Pop)', FALSE, '{"primary": "#FF204F", "bg": "#FFE8AB", "card": "#FFFDF5", "text": "#4A3B2A", "muted": "#9C8C74", "border": "#E6D5A8"}'),
-- 6. Cyber
('01938d64-5c6b-7d24-8f3a-000000000006'::uuid, 'Cyberpunk (Cyber)', TRUE, '{"primary": "#2DC8E1", "bg": "#4C2F6C", "card": "#5D3A85", "text": "#FFFFFF", "muted": "#D8B4E2", "border": "#7A5499"}')
ON CONFLICT (id) DO UPDATE SET 
    name = EXCLUDED.name, 
    is_dark = EXCLUDED.is_dark, 
    colors = EXCLUDED.colors;

-- Currencies
INSERT INTO currencies (code) VALUES
('USD'),
('EUR'),
('GBP'),
('JPY'),
('CHF'),
('CAD'),
('AUD'),
('NZD'),
('CNY'),
('RMB')
ON CONFLICT (code) DO NOTHING;

-- Account Types
INSERT INTO account_types (type, label, color, bg, icon) VALUES
(1, 'Assets', 'text-emerald-600', 'bg-emerald-100', 'Building2'),
(2, 'Liabilities', 'text-red-600', 'bg-red-100', 'CreditCard'),
(3, 'Income', 'text-blue-600', 'bg-blue-100', 'Briefcase'),
(4, 'Expenses', 'text-orange-600', 'bg-orange-100', 'Receipt')
ON CONFLICT (type) DO UPDATE SET
    label = EXCLUDED.label,
    color = EXCLUDED.color,
    bg = EXCLUDED.bg,
    icon = EXCLUDED.icon;