-- 1. 음식 종류(한식, 양식 등등)
CREATE TABLE Category (
    category_id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

-- 2. 지역 정보(시, 구)
CREATE TABLE Location (
    location_id INTEGER PRIMARY KEY,
    city TEXT NOT NULL,
    district TEXT NOT NULL,
    -- 도시 + 구 조합은 유일함
    UNIQUE(city, district)
);

-- 3. 유저 테이블
CREATE TABLE User(
    user_id INTEGER PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,

    -- 신뢰도 메타데이터
    review_count INTEGER NOT NULL DEFAULT 0,
    -- 신뢰도 점수 0.00 ~ 1.00
    reliability_score REAL NOT NULL DEFAULT .5,
    -- 극단저 평점 개수(1점 or 5점만 하는 경우가 있을 수도 있음)
    bias_count INTEGER NOT NULL DEFAULT 0,

    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now'))
);

-- 4. 식당 정보 테이블
CREATE TABLE Restaurant(
    restaurant_id INTEGER PRIMARY KEY,
    -- 외래키, 이 식당 테이블을 만든사람
    owner INTEGER NOT NULL,

    -- 정보 필드
    restaurant_name TEXT NOT NULL,
    restaurant_address TEXT NOT NULL,
    -- 외래키
    category_ref_id INTEGER NOT NULL,
    location_ref_id  INTEGER NOT NULL,

    -- 시간 메타데이터 필드
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
    last_modified_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
    last_accessed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),

    -- 외래키 연결
    FOREIGN KEY(category_ref_id) REFERENCES Category(category_id),
    FOREIGN KEY(location_ref_id) REFERENCES Location(location_id),
    FOREIGN KEY(owner) REFERENCES User(user_id)
);

-- 5. 캐싱 테이블, 중요함. 일단 검색이 되면 이 테이블을 먼저 들어가서 찾은 후
-- 이 테이블에 있는지 확인, 있으면 바로 추출, 없으면 restaurant테이블로 들어가서 찾아야 함
CREATE TABLE Cache_Metadata(
    -- 중요, 개인키이자, 외래키
    restaurant_id INTEGER PRIMARY KEY,

    -- 캐시 데이터 효율성을 위해 location/category id도 가져옴
    -- 원래는 그냥 join을 써서 찾아도 됨.
    -- 이런 식당 정보 테이블이 많아질 수록 join을 사용하는게 효율적일 것 같음
    location_ref_id INTEGER NOT NULL,
    category_ref_id INTEGER NOT NULL,

    -- 핵심 캐시 필드

    -- 가중치를 적용한 레이팅 점수(그 사람의 신뢰성을 기반으로 레이팅을 조정)
    weighted_rating REAL NOT NULL,
    -- 가중치가 적용된 리뷰의 개수
    total_weighted_reviews INTEGER NOT NULL,
    -- 캐싱이 될지 말지 결정하는 점수, 이 점수를 기반으로 캐싱 테이블에 올라오거나 내려감
    cache_score INTEGER NOT NULL,
    last_cache_updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),

    FOREIGN KEY(restaurant_id) REFERENCES Restaurant(restaurant_id),
    FOREIGN KEY(location_ref_id) REFERENCES Location(location_id),
    FOREIGN KEY(category_ref_id) REFERENCES Category(category_id)
);

-- 6. 리뷰 테이블, 실제 리뷰 데이터가 저장될 테이블
CREATE TABLE Review(
    review_id INTEGER PRIMARY KEY,

    restaurant_ref_id INTEGER NOT NULL,
    user_ref_id INTEGER NOT NULL,

    rating REAL NOT NULL,
    review_content TEXT NOT NULL,

    -- 신뢰도 시스템, 신뢰도 가중치 속성
    reliability_weight REAL NOT NULL DEFAULT .5,

    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),

    FOREIGN KEY(restaurant_ref_id) REFERENCES Restaurant(restaurant_id),
    FOREIGN KEY(user_ref_id) REFERENCES User(user_id)
);

-- 7. 리뷰 테이블에 신뢰도 변동을 요청하는 테이블, 레이팅이 될 때마다 계산하는 데 시간이 걸리니
-- 비동기적으로 변동을 계산하여 버퍼에 보내줌
CREATE TABLE Review_Analysis_Log (
    analysis_log_id INTEGER PRIMARY KEY,

    review_ref_id INTEGER NOT NULL,
    user_ref_id INTEGER NOT NULL,

    -- 신뢰도 점수를 +- 시키는 정도
    change_reliability_score REAL NOT NULL,
    new_bias_count INTEGER NOT NULL DEFAULT 0,

    log_updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
    -- 버퍼에 들어갔는지 상태
    status TEXT NOT NULL DEFAULT 'PENDING',


    FOREIGN KEY(review_ref_id) REFERENCES Review(review_id),
    FOREIGN KEY(user_ref_id) REFERENCES User(user_id)
);

-- 8. 버퍼, 일단 모든 별점 및 리뷰는 이 버퍼로 저장되고, 일정 수준의 개수가 쌓이면 다른 테이블에 적용
CREATE TABLE Buffer_Log(
    log_id INTEGER PRIMARY KEY,
    transaction_type TEXT NOT NULL, -- INSERT, UPDATE, DELETE
    target_table TEXT NOT NULL, -- 어느 테이블에 적용할지 결정하는 속성, 리뷰만 적용한다고 생각할 수 있지만, 신뢰도는 유저테이블에 있음

    payload TEXT NOT NULL, -- 페이로드는 json임, 하지만 splite에서는 text로 저장한다네요..?
    target_record_id INTEGER, -- 실제 transaction_type에 따라 적용시킬 레코드의 id

    log_updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
    is_committed INTEGER NOT NULL DEFAULT 0 -- splite에서는 boolean을 못쓴다네요..?
);

-- User: 신뢰도 점수 기반 검색 및 순위화를 위한 인덱스
CREATE INDEX idx_user_reliability_score ON User (reliability_score DESC);

-- Restaurant: 지역 및 카테고리 기반 검색을 위한 복합 인덱스
CREATE INDEX idx_restaurant_location_category ON Restaurant (Location_Ref_ID, Category_Ref_ID);

-- Buffer_Log: Worker가 미처리 로그를 효율적으로 조회하기 위한 인덱스
CREATE INDEX idx_buffer_pending ON Buffer_Log (is_committed, log_updated_at);