-- Sem partição/cluster para não mudar a spec existente
CREATE OR REPLACE TABLE `worlddata-439415.Midias_Instagram.EngajamentoInstagram` AS
WITH base AS (
  SELECT
    username                                        AS cliente_raw,
    DATE(timestamp)                                  AS data,
    media_id,
    media_type,
    media_product_type,
    permalink,
    impressions,
    reach,
    like_count,
    comments_count,
    saved,
    views,
    account_id,
    ig_id,
    owner,
    shortcode,
    media_url,
    thumbnail_url,
    caption,
    dataddo_extraction_timestamp,
    follows,
    profile_visits,
    total_interactions,
    replies,
    likes,
    LOWER(TRIM(username))                            AS uname_norm
  FROM `worlddata-439415.Midias_Instagram.postagens2`
),
ranked AS (
  SELECT
    b.*,
    ROW_NUMBER() OVER (
      PARTITION BY b.uname_norm, b.media_id, b.data
      ORDER BY b.dataddo_extraction_timestamp DESC
    ) AS rn
  FROM base b
)
SELECT
  cliente_raw AS cliente,
  CASE
    WHEN uname_norm IN ('control f5','amig','controlf5oficial','amig.bet') THEN 'Interno'
    ELSE 'Externo'
  END AS tipo_cliente,
  data,
  media_id,
  media_type,
  media_product_type,
  permalink,
  impressions,
  reach,
  like_count,
  comments_count,
  saved,
  views,
  account_id,
  ig_id,
  owner,
  shortcode,
  media_url,
  thumbnail_url,
  caption,
  dataddo_extraction_timestamp,
  follows,
  profile_visits,
  total_interactions,
  replies,
  likes,
  'Growth'         AS area,
  'Mídias Sociais' AS produto,
  'Instagram Post' AS origem
FROM ranked
WHERE rn = 1;