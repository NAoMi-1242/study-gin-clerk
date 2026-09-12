-- 000001_create_initial_tables.down.sql
-- ロールバック: 外部キー制約に配慮し逆順で削除

DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS chats;
DROP TABLE IF EXISTS user_api_keys;

