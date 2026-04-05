-- TypeFolder int = 0
-- TypeVideo  int = 1

DROP TABLE IF EXISTS objects CASCADE;

CREATE TABLE objects
(
    id          BIGSERIAL PRIMARY KEY,
    path        TEXT     NOT NULL UNIQUE,
    typ         SMALLINT NOT NULL,
    format      TEXT     NOT NULL DEFAULT '',
    file_size   BIGINT   NOT NULL DEFAULT 0,
    video_codec TEXT     NOT NULL DEFAULT '',
    audio_codec TEXT     NOT NULL DEFAULT '',
    width       INT      NOT NULL DEFAULT 0,
    height      INT      NOT NULL DEFAULT 0,
    channels    INT      NOT NULL DEFAULT 0,
    bitrate     INT      NOT NULL DEFAULT 0,
    frequency   INT      NOT NULL DEFAULT 0,
    duration    BIGINT   NOT NULL DEFAULT 0,
    bookmark    BIGINT,
    date        BIGINT   NOT NULL DEFAULT 0,
    online      BOOLEAN  NOT NULL DEFAULT true,
    dirty       BOOLEAN  NOT NULL DEFAULT true
);

CREATE OR REPLACE PROCEDURE index_add(IN is_dir BOOLEAN, IN full_path TEXT) AS
$$
DECLARE
    obj_typ   SMALLINT;
    obj_dirty BOOLEAN;
BEGIN
    IF is_dir THEN
        obj_typ := 0;
        obj_dirty := false;
    ELSE
        obj_typ := 1;
        obj_dirty := true;
    END IF;

    INSERT INTO objects (typ, path, online, dirty)
    VALUES (obj_typ, full_path, true, obj_dirty)
    ON CONFLICT(path) DO UPDATE SET typ    = EXCLUDED.typ,
                                    path   = EXCLUDED.path,
                                    online = EXCLUDED.online,
                                    dirty  = EXCLUDED.dirty;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE PROCEDURE index_delete(IN is_dir BOOLEAN, IN full_path TEXT) AS
$$
BEGIN
    DELETE FROM objects WHERE path = full_path;
    IF is_dir THEN
        DELETE FROM objects WHERE starts_with(path, CONCAT(full_path, '/'));
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE PROCEDURE index_rename(IN is_dir BOOLEAN, IN old_path TEXT, new_path TEXT) AS
$$
DECLARE
    old_path_len        INTEGER;
    old_path_with_slash TEXT;
BEGIN
    UPDATE objects SET path = new_path WHERE path = old_path;

    IF is_dir THEN
        old_path_len := length(old_path) + 2;
        old_path_with_slash := concat(old_path, '/');

        UPDATE objects
        SET path = concat(new_path, '/', SUBSTRING(path, old_path_len))
        WHERE starts_with(path, old_path_with_slash);
    END IF;
END;
$$ LANGUAGE plpgsql;
