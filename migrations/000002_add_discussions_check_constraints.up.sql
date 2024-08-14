ALTER TABLE discussions ADD CONSTRAINT movies_url_check CHECK (url ~ '^(https?:\/\/)?(www\.)?[\w-]+(\.[\w-]+)+(\/.*)?$');

ALTER TABLE discussions ADD CONSTRAINT movies_title_length_check CHECK (char_length(title) BETWEEN 1 AND 130);

ALTER TABLE discussions ADD CONSTRAINT movies_description_length_check CHECK (char_length(description) BETWEEN 1 AND 4000);
