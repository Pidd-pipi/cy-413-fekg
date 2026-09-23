-- 002_weekly_reports.sql
-- 14 天情绪周报固化快照。正常运行时由 GORM AutoMigrate 创建，本文件作为迁移脚本副本与生产表文档。
-- (user_id, end_date) 唯一索引保证同一账号同一结束日并发生成只保留一份；
-- 快照固化后，情绪记录或日记的修改/删除不会回写到本表。
CREATE TABLE IF NOT EXISTS weekly_reports (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    average_mood DOUBLE PRECISION NOT NULL DEFAULT 0,
    lowest_date DATE,
    lowest_level INT NOT NULL DEFAULT 0,
    valid_days INT NOT NULL DEFAULT 0,
    days TEXT NOT NULL,
    tag_counts TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_weekly_report_user_end
    ON weekly_reports (user_id, end_date);
