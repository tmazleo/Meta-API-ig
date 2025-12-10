-- Tabela destino: worlddata-439415.Midias_Instagram.EngajamnetoStories
CREATE OR REPLACE TABLE `worlddata-439415.Midias_Instagram.EngajamnetoStories` AS
WITH base AS (
  SELECT
    -- Normalizações
    LOWER(TRIM(username))                 AS uname_norm,
    username                              AS cliente,
    CASE
      WHEN LOWER(TRIM(username)) IN ('controlf5oficial','amig.bet','control f5','amig') THEN 'Interno'
      ELSE 'Externo'
    END                                   AS tipo_cliente,

    -- Datas
    DATE(timestamp)                       AS data,            -- data real do story
    TIMESTAMP(timestamp)                  AS ts_story,        -- timestamp original (se quiser usar em análises)
    DATE(date)                            AS insert_date,     -- data entregue pelo conector (se existir/for útil)

    -- Identificadores / metadados do story
    media_id,
    account_id,
    ig_id,
    owner,
    shortcode,
    media_url,
    permalink,
    thumbnail_url,
    caption,
    media_type,

    -- Métricas do story
    SAFE_CAST(exits        AS INT64)      AS exits,
    SAFE_CAST(impressions  AS INT64)      AS impressions,
    SAFE_CAST(reach        AS INT64)      AS reach,
    SAFE_CAST(replies      AS INT64)      AS replies,
    SAFE_CAST(taps_forward AS INT64)      AS taps_forward,
    SAFE_CAST(taps_back    AS INT64)      AS taps_back

  FROM `worlddata-439415.Midias_Instagram.Story`
),
ranked AS (
  SELECT
    b.*,
    ROW_NUMBER() OVER (
      PARTITION BY b.uname_norm, b.media_id, b.data
      ORDER BY ts_story DESC   -- mantém o snapshot mais recente do dia
    ) AS rn
  FROM base b
)
SELECT
  cliente,
  tipo_cliente,
  data,
  media_id,
  account_id,
  ig_id,
  owner,
  shortcode,
  media_url,
  permalink,
  thumbnail_url,
  caption,
  media_type,
  exits,
  impressions,
  reach,
  replies,
  taps_forward,
  taps_back,

  -- Metadados fixos p/ consolidação
  'Growth'          AS area,
  'Mídias Sociais'  AS produto,
  'Instagram Stories' AS origem
FROM ranked
WHERE rn = 1;
