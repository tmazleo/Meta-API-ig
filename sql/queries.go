package sql

const (
	MergePerfil = `
    MERGE ` + "`worlddata-439415.Midias_Instagram.Instagram_perfil_tratado`" + ` T
    USING (
        SELECT * EXCEPT(row_num)
        FROM (
            SELECT
                DATE(date) AS data,
                id, ig_id, username,
                COALESCE(username, id) AS cliente,
                -- Garantindo que o tipo_cliente seja consistente
                CASE 
                    WHEN LOWER(TRIM(username)) IN ('controlf5oficial','amig.bet') THEN 'Interno' 
                    ELSE 'Externo' 
                END AS tipo_cliente,
                -- Tratando possíveis nulos para o Looker não mostrar traços
                COALESCE(followers_count, 0) AS followers_count, 
                COALESCE(follows_count, 0) AS follows_count, 
                COALESCE(media_count, 0) AS media_count, 
                profile_picture_url,
                extraction_timestamp,
                'Growth' AS area, 
                'Mídias Sociais' AS produto, 
                'Instagram Perfil' AS origem,
                ROW_NUMBER() OVER(PARTITION BY DATE(date), ig_id ORDER BY extraction_timestamp DESC) AS row_num
            FROM ` + "`worlddata-439415.Midias_Instagram.perfil_meta`" + `
        ) WHERE row_num = 1
    ) S
    ON T.data = S.data AND T.ig_id = S.ig_id
    WHEN MATCHED THEN
        UPDATE SET 
            T.followers_count = S.followers_count, 
            T.follows_count = S.follows_count,
            T.media_count = S.media_count, 
            T.username = S.username,
            T.tipo_cliente = S.tipo_cliente,
            T.profile_picture_url = S.profile_picture_url,
            T.extraction_timestamp = S.extraction_timestamp
    WHEN NOT MATCHED THEN
        INSERT (data, id, ig_id, username, cliente, tipo_cliente, followers_count, follows_count, media_count, profile_picture_url, area, produto, origem, extraction_timestamp)
        VALUES (S.data, S.id, S.ig_id, S.username, S.cliente, S.tipo_cliente, S.followers_count, S.follows_count, S.media_count, S.profile_picture_url, S.area, S.produto, S.origem, S.extraction_timestamp);`

	MergePostagens = `
    MERGE ` + "`worlddata-439415.Midias_Instagram.Instagram_postagens_tratado`" + ` T
    USING (
        SELECT * EXCEPT(row_num, uname_norm)
        FROM (
            SELECT
                username AS cliente,
                CASE 
                    WHEN LOWER(TRIM(username)) IN ('controlf5oficial','amig.bet') THEN 'Interno' 
                    ELSE 'Externo' 
                END AS tipo_cliente,
                -- CONVERSÃO PARA DATA E FUSO BRASÍLIA
                DATE(SAFE_CAST(timestamp AS TIMESTAMP), "America/Sao_Paulo") AS data,
                -- TIMESTAMP COMPLETO PARA O LOOKER (DIMENSÃO DE HORA)
                DATETIME(SAFE_CAST(timestamp AS TIMESTAMP), "America/Sao_Paulo") AS data_hora,
                media_id, media_type, media_product_type, permalink,
                
                -- Proteção para métricas com MAX e tratamento de NULLs
                MAX(COALESCE(impressions, 0)) OVER(PARTITION BY media_id) AS impressions,
                MAX(COALESCE(reach, 0)) OVER(PARTITION BY media_id) AS reach,
                MAX(COALESCE(like_count, 0)) OVER(PARTITION BY media_id) AS like_count,
                MAX(COALESCE(comments_count, 0)) OVER(PARTITION BY media_id) AS comments_count,
                MAX(COALESCE(replies, 0)) OVER(PARTITION BY media_id) AS replies,
                MAX(COALESCE(shares, 0)) OVER(PARTITION BY media_id) AS shares,
                MAX(COALESCE(saved, 0)) OVER(PARTITION BY media_id) AS saved,
                MAX(COALESCE(views, 0)) OVER(PARTITION BY media_id) AS views,

                -- CÁLCULO DO TOTAL INTERAÇÕES (Soma de todos os engajamentos)
                MAX(
                    COALESCE(like_count, 0) + 
                    COALESCE(comments_count, 0) + 
                    COALESCE(replies, 0) + 
                    COALESCE(shares, 0) + 
                    COALESCE(saved, 0)
                ) OVER(PARTITION BY media_id) AS total_interactions,

                account_id,
                account_id AS ig_id,
                owner, shortcode, media_url, thumbnail_url, caption,
                SAFE_CAST(dataddo_extraction_timestamp AS TIMESTAMP) AS dataddo_extraction_timestamp,
                'Growth' AS area,
                'Mídias Sociais' AS produto,
                'Instagram Post' AS origem,
                LOWER(TRIM(username)) AS uname_norm,
                ROW_NUMBER() OVER (
                    PARTITION BY media_id 
                    ORDER BY SAFE_CAST(dataddo_extraction_timestamp AS TIMESTAMP) DESC
                ) AS row_num
            FROM ` + "`worlddata-439415.Midias_Instagram.postagens_meta`" + `
        ) WHERE row_num = 1
    ) S
    ON T.media_id = S.media_id
    WHEN MATCHED THEN
        UPDATE SET 
            T.cliente = S.cliente,
            T.tipo_cliente = S.tipo_cliente,
            T.data = S.data,
            T.data_hora = S.data_hora,
            T.like_count = S.like_count, 
            T.comments_count = S.comments_count,
            T.replies = S.replies,
            T.shares = S.shares,
            T.total_interactions = S.total_interactions,
            T.impressions = S.impressions,
            T.reach = S.reach, 
            T.saved = S.saved, 
            T.views = S.views,
            T.media_url = S.media_url,
            T.thumbnail_url = S.thumbnail_url,
            T.caption = S.caption,
            T.dataddo_extraction_timestamp = S.dataddo_extraction_timestamp
    WHEN NOT MATCHED THEN
        INSERT (
            cliente, tipo_cliente, data, data_hora, media_id, media_type, media_product_type, 
            permalink, impressions, reach, like_count, comments_count, replies, shares, total_interactions, saved, 
            views, account_id, ig_id, owner, shortcode, media_url, 
            thumbnail_url, caption, dataddo_extraction_timestamp, area, produto, origem
        )
        VALUES (
            S.cliente, S.tipo_cliente, S.data, S.data_hora, S.media_id, S.media_type, S.media_product_type, 
            S.permalink, S.impressions, S.reach, S.like_count, S.comments_count, S.replies, S.shares, S.total_interactions, S.saved, 
            S.views, S.account_id, S.ig_id, S.owner, S.shortcode, S.media_url, 
            S.thumbnail_url, S.caption, S.dataddo_extraction_timestamp, S.area, S.produto, S.origem
        );`

	MergeStories = `
	MERGE ` + "`worlddata-439415.Midias_Instagram.Instagram_story_tratado`" + ` T
	USING (
		SELECT * EXCEPT(row_num, uname_norm)
		FROM (
			SELECT
				-- 1. cliente
				username AS cliente,
				-- 2. tipo_cliente
				CASE 
					WHEN LOWER(TRIM(username)) IN ('controlf5oficial','amig.bet') THEN 'Interno' 
					ELSE 'Externo' 
				END AS tipo_cliente,
				-- 3. data
				DATE(timestamp) AS data,
				-- 4. media_id
				media_id,
				-- 5. account_id
				account_id,
				-- 6. ig_id
				account_id AS ig_id,
				-- 7. owner
				owner,
				-- 8. shortcode (Não existe na meta, preenchendo vazio)
				'' AS shortcode,
				-- 9. media_url
				media_url,
				-- 10. permalink (Não existe na meta, preenchendo vazio)
				'' AS permalink,
				-- 11. thumbnail_url (Não existe na meta, preenchendo vazio)
				'' AS thumbnail_url,
				-- 12. caption (Não existe na meta, preenchendo vazio)
				'' AS caption,
				-- 13. media_type
				media_type,
				-- 14. exits
				exits,
				-- 15. impressions
				impressions,
				-- 16. reach
				reach,
				-- 17. replies
				replies,
				-- 18. taps_forward
				taps_forward,
				-- 19. taps_back
				taps_back,
				-- 20. area
				'Growth' AS area,
				-- 21. produto
				'Mídias Sociais' AS produto,
				-- 22. origem
				'Instagram Stories' AS origem,
				-- Campos para lógica interna
				LOWER(TRIM(username)) AS uname_norm,
				-- Usando 'timestamp' para ordenação, já que 'extraction_timestamp' não existe na meta
				ROW_NUMBER() OVER (
					PARTITION BY media_id 
					ORDER BY timestamp DESC
				) AS row_num
			FROM ` + "`worlddata-439415.Midias_Instagram.story_meta`" + `
		) WHERE row_num = 1
	) S
	ON T.media_id = S.media_id
	WHEN MATCHED THEN
		UPDATE SET 
			T.exits = S.exits, 
			T.impressions = S.impressions, 
			T.reach = S.reach, 
			T.replies = S.replies, 
			T.taps_forward = S.taps_forward, 
			T.taps_back = S.taps_back
	WHEN NOT MATCHED THEN
		-- Mapeamento explícito de todas as 22 colunas identificadas no seu print de destino
		INSERT (
			cliente, tipo_cliente, data, media_id, account_id, ig_id, owner, 
			shortcode, media_url, permalink, thumbnail_url, caption, media_type, 
			exits, impressions, reach, replies, taps_forward, taps_back, 
			area, produto, origem
		)
		VALUES (
			S.cliente, S.tipo_cliente, S.data, S.media_id, S.account_id, S.ig_id, S.owner, 
			S.shortcode, S.media_url, S.permalink, S.thumbnail_url, S.caption, S.media_type, 
			S.exits, S.impressions, S.reach, S.replies, S.taps_forward, S.taps_back, 
			S.area, S.produto, S.origem
		);`
)
