--------------------
-- EPISODES
--------------------
CREATE TABLE episodes_v1 (
    id character varying(128) PRIMARY KEY,
    media_id character varying(128) NOT NULL REFERENCES media(id),
    absolute_number integer NOT NULL,
    description text NOT NULL,
    title text NOT NULL,
    air_date date,
    season double precision NOT NULL,
    season_number double precision NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    UNIQUE (media_id, season, season_number)
);
COMMENT ON COLUMN episodes_v1.id IS 'ID this episode, a UUID';
COMMENT ON COLUMN episodes_v1.media_id IS 'ID of the media this episode belong too';
COMMENT ON COLUMN episodes_v1.absolute_number IS 'Absolute episode number';
COMMENT ON COLUMN episodes_v1.description IS 'Synopsis, or description, of this episode';
COMMENT ON COLUMN episodes_v1.air_date IS 'When this episode first aired, this is only acurrate to the MM-DD-YYYY';
COMMENT ON COLUMN episodes_v1.created_at IS 'When this episode was added';
COMMENT ON COLUMN episodes_v1.season IS 'Season this episode was in';
COMMENT ON COLUMN episodes_v1.title IS 'Episode title';
COMMENT ON COLUMN episodes_v1.season_number IS 'Number this episode was in a season';

--------------------
-- EPISODE FILES
--------------------
CREATE TABLE episode_files_v1 (
    id character varying(128) PRIMARY KEY,
    episode_id character varying(128) NOT NULL REFERENCES episodes_v1(id),
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
-- SERIES DATA
--------------------
CREATE TABLE series_v1 (
    id character varying(128) PRIMARY KEY REFERENCES media(id),
    title text NOT NULL,
    type integer NOT NULL,
    rating double precision DEFAULT '10'::double precision,
    overview text NOT NULL,
    network text,
    first_aired date NOT NULL,
    status text NOT NULL,
    genres text NOT NULL,
    airs text NOT NULL,
    air_day_of_week text NOT NULL,
    runtime integer DEFAULT 0,
    created_at timestamp with time zone DEFAULT now()
);
COMMENT ON COLUMN series_v1.id IS 'ID of the media file';
COMMENT ON COLUMN series_v1.title IS 'Title of the Media';
COMMENT ON COLUMN series_v1.type IS 'Type of the Media';
COMMENT ON COLUMN series_v1.rating IS 'Rating of Media, 0-10';
COMMENT ON COLUMN series_v1.overview IS 'Overview / description of the media';
COMMENT ON COLUMN series_v1.network IS 'Network this media played on';
COMMENT ON COLUMN series_v1.first_aired IS 'When this media first started airing';
COMMENT ON COLUMN series_v1.status IS 'Status of this media, is it still airing?';
COMMENT ON COLUMN series_v1.genres IS 'CSV list of genres this media is';
COMMENT ON COLUMN series_v1.airs IS 'HH:MM time that this show airs';
COMMENT ON COLUMN series_v1.air_day_of_week IS 'Day of the week this show airs';
COMMENT ON COLUMN series_v1.runtime IS 'Runtime of this media on average';

--------------------
-- IMAGES
--------------------
CREATE TABLE images_v1 (
    id character varying(128) PRIMARY KEY,
    media_id character varying(128) NOT NULL REFERENCES media(id),
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


-------------------
-- EPSIODE IMAGES
-------------------
CREATE TABLE episode_images_v1 (
    id character varying(128) PRIMARY KEY,
    episode_id character varying(128) NOT NULL REFERENCES episodes_v1(id),
    checksum text NOT NULL,
    image_type text NOT NULL,
    rating double precision NOT NULL,
    resolution text NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    UNIQUE (episode_id, checksum)
);
COMMENT ON COLUMN episode_images_v1.id IS 'ID of this image file';
COMMENT ON COLUMN episode_images_v1.episode_id IS 'ID of the media this image is associated with';
COMMENT ON COLUMN episode_images_v1.image_type IS 'Type of image: background/poster';
COMMENT ON COLUMN episode_images_v1.rating IS 'Rating of this image, defaults to 10 if provider didnt provide one';
COMMENT ON COLUMN episode_images_v1.resolution IS 'Media resolution, in WxH';
COMMENT ON COLUMN episode_images_v1.created_at IS 'When this was created';
