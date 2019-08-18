--------------------
-- EPISODE FILES
--------------------
CREATE TABLE episode_files_v1 (
    id character varying(128) PRIMARY KEY,
    episode_id character varying(128) NOT NULL,
    key text NOT NULL,
    quality text,
    created_at timestamp with time zone DEFAULT now()
);
COMMENT ON COLUMN episode_files_v1.id IS 'ID of this file';
COMMENT ON COLUMN episode_files_v1.episode_id IS 'Episode this file is associated with';
COMMENT ON COLUMN episode_files_v1.key IS 'S3 key of this file';
COMMENT ON COLUMN episode_files_v1.quality IS 'Quality of this file, valid options are: 480p, 720p, 1080p, 2k, 4k';
COMMENT ON COLUMN episode_files_v1.created_at IS 'When this file was added';

--------------------
-- EPISODES
--------------------
CREATE TABLE episodes_v1 (
    id character varying(128) PRIMARY KEY,
    media_id character varying(128) NOT NULL,
    absolute_number integer NOT NULL,
    description text NOT NULL,
    air_date date,
    season double precision NOT NULL,
    season_number double precision NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    UNIQUE (media_id, season, season_number),
    UNIQUE (media_id, absolute_number)
);
COMMENT ON COLUMN episodes_v1.id IS 'ID this episode, a UUID';
COMMENT ON COLUMN episodes_v1.media_id IS 'ID of the media this episode belong too';
COMMENT ON COLUMN episodes_v1.absolute_number IS 'Absolute episode number';
COMMENT ON COLUMN episodes_v1.description IS 'Synopsis, or description, of this episode';
COMMENT ON COLUMN episodes_v1.air_date IS 'When this episode first aired, this is only acurrate to the MM-DD-YYYY';
COMMENT ON COLUMN episodes_v1.created_at IS 'When this episode was added';
COMMENT ON COLUMN episodes_v1.season IS 'Season this episode was in';
COMMENT ON COLUMN episodes_v1.season_number IS 'Number this episode was in a season';

--------------------
-- IMAGES
--------------------
CREATE TABLE images_v1 (
    id character varying(128) PRIMARY KEY,
    media_id character varying(128) NOT NULL,
    checksum text NOT NULL,
    image_type text NOT NULL,
    rating float NOT NULL,
    resolution text NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    UNIQUE (media_id, checksum)
);
COMMENT ON COLUMN images_v1.id IS 'ID of this image file';
COMMENT ON COLUMN images_v1.media_id IS 'ID of the media this image is associated with';
COMMENT ON COLUMN images_v1.image_type IS 'Type of image: background/poster';
COMMENT ON COLUMN images_v1.rating IS 'Rating of this image, defaults to 10 if provider didnt provide one';
COMMENT ON COLUMN images_v1.created_at IS 'When this was created';
COMMENT ON COLUMN images_v1.resolution IS 'Media resolution, in WxH';
