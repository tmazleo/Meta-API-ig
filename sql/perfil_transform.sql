CREATE OR REPLACE TABLE `worlddata-439415.Midias_Instagram.EngajamentoInstagram_Perfil` AS 
SELECT
  DATE(date)                AS data,         -- Padroniza a data para formato DATE
  id,
  ig_id,
  username,
  name                     AS cliente,      -- Identificador do cliente
  CASE 
    WHEN username IN ('controlf5oficial','amig.bet') THEN 'Interno'
    ELSE 'Externo'
  END                       AS tipo_cliente, -- Novo campo categorizando o cliente
  followers_count,
  follows_count,
  media_count,
  profile_picture_url,
  website,
  follower_count,
  impressions,
  reach,
  DATE(dataddo_insert_date) AS insert_date,  -- Data de coleta
  'Growth'                 AS area,         -- Área no dashboard
  'Mídias Sociais'         AS produto,      -- Produto dentro de Growth
  'Instagram Perfil'       AS origem        -- Fonte dos dados
FROM 
  `worlddata-439415.Midias_Instagram.perfil`;
