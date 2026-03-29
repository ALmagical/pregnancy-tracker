CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    openid TEXT NOT NULL UNIQUE,
    unionid TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pregnant',
    last_period_date DATE,
    due_date DATE,
    pre_pregnancy_weight NUMERIC(6,1),
    height_cm NUMERIC(6,1),
    current_weight NUMERIC(6,1),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS checkups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    checkup_date DATE NOT NULL,
    checkup_type TEXT NOT NULL,
    checkup_type_id TEXT NOT NULL DEFAULT '',
    hospital TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_checkups_user_date ON checkups(user_id, checkup_date DESC);

CREATE TABLE IF NOT EXISTS checkup_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    checkup_id UUID NOT NULL REFERENCES checkups(id) ON DELETE CASCADE,
    storage_key TEXT NOT NULL,
    public_url TEXT NOT NULL,
    thumb_url TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_reports_checkup ON checkup_reports(checkup_id);

CREATE TABLE IF NOT EXISTS weights (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    weight NUMERIC(6,1) NOT NULL,
    recorded_at DATE NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    week SMALLINT,
    day SMALLINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_weights_user ON weights(user_id, recorded_at DESC);

CREATE TABLE IF NOT EXISTS fm_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'running',
    count INT NOT NULL DEFAULT 0,
    result_tag TEXT,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_fm_user_started ON fm_sessions(user_id, started_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_fm_one_running_per_user ON fm_sessions(user_id) WHERE status = 'running';

CREATE TABLE IF NOT EXISTS contractions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ NOT NULL,
    duration_sec INT NOT NULL,
    interval_sec INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_contractions_user ON contractions(user_id, started_at DESC);

CREATE TABLE IF NOT EXISTS checklist_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id TEXT NOT NULL,
    title TEXT NOT NULL,
    checked BOOLEAN NOT NULL DEFAULT false,
    note TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT 'template',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_checklist_user ON checklist_items(user_id, sort_order);

CREATE TABLE IF NOT EXISTS pregnancy_week_tasks (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    week INT NOT NULL,
    task_id TEXT NOT NULL,
    done BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (user_id, week, task_id)
);

CREATE TABLE IF NOT EXISTS articles (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    cover TEXT NOT NULL DEFAULT '',
    tags TEXT[] NOT NULL DEFAULT '{}',
    read_minutes INT NOT NULL DEFAULT 5,
    content TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    published_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS favorites (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, target_type, target_id)
);

CREATE TABLE IF NOT EXISTS user_settings (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    push JSONB NOT NULL DEFAULT '{}',
    ai JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS export_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    types TEXT[] NOT NULL,
    format TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'processing',
    file_path TEXT,
    public_url TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_export_user_created ON export_jobs(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS idempotency_records (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    idem_key TEXT NOT NULL,
    response_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, idem_key)
);

INSERT INTO articles (id, title, summary, cover, tags, read_minutes, content, source, published_at) VALUES
('a_1001', '大排畸怎么做', '检查目的、流程与注意事项，帮你轻松应对超声大排畸。', '', ARRAY['产检']::TEXT[], 5,
 '<p>大排畸一般在孕 20–24 周进行，用于系统观察胎儿结构发育情况。</p><p>检查前无需空腹，按医嘱准备即可。</p>',
 '科普整理', now() - interval '30 days'),
('a_1002', 'B超单怎么看', '常见指标含义与需要关注的提示，避免过度解读。', '', ARRAY['产检']::TEXT[], 4,
 '<p>B超单中的双顶径、腹围、股骨长等用于评估胎儿生长趋势。</p><p>具体解读请以产检医生意见为准。</p>',
 '科普整理', now() - interval '25 days'),
('a_1003', '孕中期营养要点', '均衡饮食与体重管理的基础建议。', '', ARRAY['营养']::TEXT[], 6,
 '<p>适量增加优质蛋白与膳食纤维，注意补铁补钙遵医嘱。</p>',
 '科普整理', now() - interval '20 days'),
('a_1004', '胎动计数怎么数', '简单可行的居家胎动记录方法。', '', ARRAY['胎动']::TEXT[], 3,
 '<p>可在相对固定时间安静环境下计数，记录趋势比单次绝对值更重要。</p>',
 '科普整理', now() - interval '15 days')
ON CONFLICT (id) DO NOTHING;
