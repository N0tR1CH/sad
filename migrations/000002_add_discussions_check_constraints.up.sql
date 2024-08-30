ALTER TABLE discussions ADD CONSTRAINT discussions_url CHECK (url ~ '^(https?:\/\/)?(www\.)?[\w-]+(\.[\w-]+)+(\/.*)?$');

ALTER TABLE discussions ADD CONSTRAINT discussions_title CHECK (char_length(title) BETWEEN 1 AND 130);

ALTER TABLE discussions ADD CONSTRAINT discussions_description CHECK (char_length(description) BETWEEN 1 AND 4000);
