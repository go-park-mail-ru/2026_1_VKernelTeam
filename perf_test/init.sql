--
-- PostgreSQL database dump
--

\restrict iuvlNFybJO1pdAjHhfh5LXc1kbqSe0aaLO5W2NCNTY2pevMBiTADibYJeH7cIAZ

-- Dumped from database version 17.9
-- Dumped by pg_dump version 17.9

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: pg_trgm; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;


--
-- Name: EXTENSION pg_trgm; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION pg_trgm IS 'text similarity measurement and index searching based on trigrams';


--
-- Name: recalc_user_review_aggregates(bigint); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.recalc_user_review_aggregates(p_user_id bigint) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE "user"
    SET rating = COALESCE((
            SELECT AVG(rating)::numeric(3,2) FROM review WHERE receiver_id = p_user_id
        ), 0),
        reviews_count = (SELECT COUNT(*) FROM review WHERE receiver_id = p_user_id)
    WHERE id = p_user_id;
END;
$$;


ALTER FUNCTION public.recalc_user_review_aggregates(p_user_id bigint) OWNER TO postgres;

--
-- Name: trg_review_aggregates(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.trg_review_aggregates() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM recalc_user_review_aggregates(NEW.receiver_id);
    ELSIF TG_OP = 'DELETE' THEN
        PERFORM recalc_user_review_aggregates(OLD.receiver_id);
    ELSIF TG_OP = 'UPDATE' THEN
        PERFORM recalc_user_review_aggregates(NEW.receiver_id);
        IF NEW.receiver_id <> OLD.receiver_id THEN
            PERFORM recalc_user_review_aggregates(OLD.receiver_id);
        END IF;
    END IF;
    RETURN NULL;
END;
$$;


ALTER FUNCTION public.trg_review_aggregates() OWNER TO postgres;

--
-- Name: update_cart_count(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.update_cart_count() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE "user" SET cart_count = cart_count + 1 WHERE id = NEW.user_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE "user" SET cart_count = cart_count - 1 WHERE id = OLD.user_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$;


ALTER FUNCTION public.update_cart_count() OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: cart_item; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.cart_item (
    user_id bigint NOT NULL,
    product_id bigint NOT NULL,
    quantity integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT cart_item_quantity_check CHECK ((quantity > 0))
);


ALTER TABLE public.cart_item OWNER TO postgres;

--
-- Name: category; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.category (
    id bigint NOT NULL,
    name text NOT NULL,
    parent_id bigint,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT category_name_check CHECK (((length(name) >= 2) AND (length(name) <= 100)))
);


ALTER TABLE public.category OWNER TO postgres;

--
-- Name: category_characteristic; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.category_characteristic (
    id bigint NOT NULL,
    category_id bigint NOT NULL,
    name text NOT NULL,
    allowed_values text[],
    sort_order integer DEFAULT 0 NOT NULL,
    CONSTRAINT category_characteristic_name_check CHECK (((length(name) >= 1) AND (length(name) <= 100)))
);


ALTER TABLE public.category_characteristic OWNER TO postgres;

--
-- Name: category_characteristic_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.category_characteristic ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.category_characteristic_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: category_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.category ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.category_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: chat; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.chat (
    id bigint NOT NULL,
    product_id bigint NOT NULL,
    buyer_id bigint NOT NULL,
    seller_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT chat_check CHECK ((buyer_id <> seller_id))
);


ALTER TABLE public.chat OWNER TO postgres;

--
-- Name: chat_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.chat ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.chat_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: favorite; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.favorite (
    user_id bigint NOT NULL,
    product_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


ALTER TABLE public.favorite OWNER TO postgres;

--
-- Name: message; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.message (
    id bigint NOT NULL,
    chat_id bigint NOT NULL,
    sender_id bigint NOT NULL,
    text_content text NOT NULL,
    is_read boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    msg_type text DEFAULT 'text'::text,
    CONSTRAINT message_text_content_check CHECK ((length(text_content) > 0))
);


ALTER TABLE public.message OWNER TO postgres;

--
-- Name: message_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.message ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.message_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: order; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public."order" (
    id bigint NOT NULL,
    buyer_id bigint NOT NULL,
    total_amount bigint NOT NULL,
    status text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    source text NOT NULL,
    chat_id bigint,
    CONSTRAINT order_source_check CHECK ((source = ANY (ARRAY['cart'::text, 'chat'::text]))),
    CONSTRAINT order_status_check CHECK ((status = ANY (ARRAY['created'::text, 'paid'::text, 'delivering'::text, 'completed'::text, 'cancelled'::text]))),
    CONSTRAINT order_total_amount_check CHECK ((total_amount >= 0))
);


ALTER TABLE public."order" OWNER TO postgres;

--
-- Name: order_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public."order" ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.order_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: order_item; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.order_item (
    order_id bigint NOT NULL,
    product_id bigint NOT NULL,
    price_at_purchase bigint NOT NULL,
    quantity integer DEFAULT 1 NOT NULL,
    CONSTRAINT order_item_price_at_purchase_check CHECK ((price_at_purchase >= 0)),
    CONSTRAINT order_item_quantity_check CHECK ((quantity > 0))
);


ALTER TABLE public.order_item OWNER TO postgres;

--
-- Name: payment; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.payment (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    amount bigint NOT NULL,
    status text NOT NULL,
    provider text DEFAULT 'mock'::text NOT NULL,
    provider_ref text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    confirmation_url text,
    raw_provider_payload jsonb,
    CONSTRAINT payment_amount_check CHECK ((amount > 0)),
    CONSTRAINT payment_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'succeeded'::text, 'failed'::text, 'cancelled'::text])))
);


ALTER TABLE public.payment OWNER TO postgres;

--
-- Name: payment_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.payment ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.payment_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: platform_setting; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.platform_setting (
    key text NOT NULL,
    value text NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by bigint
);


ALTER TABLE public.platform_setting OWNER TO postgres;

--
-- Name: product; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.product (
    id bigint NOT NULL,
    seller_id bigint NOT NULL,
    category_id bigint NOT NULL,
    title text NOT NULL,
    description text NOT NULL,
    price bigint NOT NULL,
    status text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at timestamp with time zone,
    location text,
    views_count bigint DEFAULT 0 NOT NULL,
    lat double precision,
    lon double precision,
    deleted_by_admin_id bigint,
    rejection_reason text,
    CONSTRAINT product_description_check CHECK (((length(description) >= 10) AND (length(description) <= 5000))),
    CONSTRAINT product_lat_range CHECK (((lat IS NULL) OR ((lat >= ('-90'::integer)::double precision) AND (lat <= (90)::double precision)))),
    CONSTRAINT product_lon_range CHECK (((lon IS NULL) OR ((lon >= ('-180'::integer)::double precision) AND (lon <= (180)::double precision)))),
    CONSTRAINT product_price_check CHECK ((price >= 0)),
    CONSTRAINT product_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'active'::text, 'reserved'::text, 'sold'::text, 'archived'::text, 'admin_deleted'::text, 'pending_moderation'::text, 'rejected'::text]))),
    CONSTRAINT product_title_check CHECK (((length(title) >= 5) AND (length(title) <= 150)))
);


ALTER TABLE public.product OWNER TO postgres;

--
-- Name: product_characteristic; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.product_characteristic (
    id bigint NOT NULL,
    product_id bigint NOT NULL,
    category_characteristic_id bigint NOT NULL,
    value text NOT NULL,
    CONSTRAINT product_characteristic_value_check CHECK (((length(value) >= 1) AND (length(value) <= 500)))
);


ALTER TABLE public.product_characteristic OWNER TO postgres;

--
-- Name: product_characteristic_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.product_characteristic ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.product_characteristic_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: product_custom_characteristic; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.product_custom_characteristic (
    id bigint NOT NULL,
    product_id bigint NOT NULL,
    name text NOT NULL,
    value text NOT NULL,
    CONSTRAINT product_custom_characteristic_name_check CHECK (((length(name) >= 1) AND (length(name) <= 100))),
    CONSTRAINT product_custom_characteristic_value_check CHECK (((length(value) >= 1) AND (length(value) <= 500)))
);


ALTER TABLE public.product_custom_characteristic OWNER TO postgres;

--
-- Name: product_custom_characteristic_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.product_custom_characteristic ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.product_custom_characteristic_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: product_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.product ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.product_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: product_image; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.product_image (
    id bigint NOT NULL,
    product_id bigint NOT NULL,
    file_path text NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


ALTER TABLE public.product_image OWNER TO postgres;

--
-- Name: product_image_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.product_image ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.product_image_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: product_price_history; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.product_price_history (
    id bigint NOT NULL,
    product_id bigint NOT NULL,
    old_price bigint NOT NULL,
    new_price bigint NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


ALTER TABLE public.product_price_history OWNER TO postgres;

--
-- Name: product_price_history_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.product_price_history ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.product_price_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: product_status_history; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.product_status_history (
    id bigint NOT NULL,
    product_id bigint NOT NULL,
    old_status text NOT NULL,
    new_status text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


ALTER TABLE public.product_status_history OWNER TO postgres;

--
-- Name: product_status_history_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.product_status_history ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.product_status_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: product_view; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.product_view (
    id bigint NOT NULL,
    product_id bigint NOT NULL,
    user_id bigint,
    viewed_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


ALTER TABLE public.product_view OWNER TO postgres;

--
-- Name: product_view_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.product_view ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.product_view_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: promotion; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.promotion (
    id bigint NOT NULL,
    product_id bigint NOT NULL,
    user_id bigint NOT NULL,
    plan_id bigint NOT NULL,
    kind text NOT NULL,
    starts_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    price_paid bigint NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT promotion_kind_check CHECK ((kind = ANY (ARRAY['boost'::text, 'highlight'::text]))),
    CONSTRAINT promotion_price_paid_check CHECK ((price_paid >= 0))
);


ALTER TABLE public.promotion OWNER TO postgres;

--
-- Name: promotion_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.promotion ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.promotion_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: promotion_plan; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.promotion_plan (
    id bigint NOT NULL,
    code text NOT NULL,
    kind text NOT NULL,
    duration_days integer NOT NULL,
    price bigint NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT promotion_plan_code_check CHECK (((length(code) >= 1) AND (length(code) <= 64))),
    CONSTRAINT promotion_plan_duration_days_check CHECK ((duration_days > 0)),
    CONSTRAINT promotion_plan_kind_check CHECK ((kind = ANY (ARRAY['boost'::text, 'highlight'::text]))),
    CONSTRAINT promotion_plan_price_check CHECK ((price >= 0))
);


ALTER TABLE public.promotion_plan OWNER TO postgres;

--
-- Name: promotion_plan_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.promotion_plan ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.promotion_plan_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: refresh_token; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.refresh_token (
    token text NOT NULL,
    user_id bigint NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


ALTER TABLE public.refresh_token OWNER TO postgres;

--
-- Name: review; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.review (
    id bigint NOT NULL,
    sender_id bigint NOT NULL,
    receiver_id bigint NOT NULL,
    product_id bigint NOT NULL,
    rating integer NOT NULL,
    content text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT review_check CHECK ((sender_id <> receiver_id)),
    CONSTRAINT review_content_check CHECK (((length(content) >= 5) AND (length(content) <= 2000))),
    CONSTRAINT review_rating_check CHECK (((rating >= 1) AND (rating <= 5)))
);


ALTER TABLE public.review OWNER TO postgres;

--
-- Name: review_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.review ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.review_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO postgres;

--
-- Name: support_message; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.support_message (
    id bigint NOT NULL,
    ticket_id bigint NOT NULL,
    user_id bigint NOT NULL,
    text text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT support_message_text_check CHECK ((length(text) > 0))
);


ALTER TABLE public.support_message OWNER TO postgres;

--
-- Name: support_message_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.support_message ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.support_message_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: support_ticket; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.support_ticket (
    id integer NOT NULL,
    user_id integer NOT NULL,
    category character varying(50) NOT NULL,
    status character varying(50) DEFAULT 'open'::character varying NOT NULL,
    title character varying(255) NOT NULL,
    description text NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    rating smallint,
    CONSTRAINT support_ticket_rating_check CHECK (((rating IS NULL) OR ((rating >= 1) AND (rating <= 5))))
);


ALTER TABLE public.support_ticket OWNER TO postgres;

--
-- Name: support_ticket_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.support_ticket_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.support_ticket_id_seq OWNER TO postgres;

--
-- Name: support_ticket_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.support_ticket_id_seq OWNED BY public.support_ticket.id;


--
-- Name: user; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public."user" (
    id bigint NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    first_name text NOT NULL,
    second_name text,
    avatar_path text,
    rating numeric DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    cart_count integer DEFAULT 0 NOT NULL,
    role character varying(20) DEFAULT 'user'::character varying NOT NULL,
    reviews_count integer DEFAULT 0 NOT NULL,
    CONSTRAINT user_email_check CHECK ((email ~* '^[A-Za-z0-9._+%-]+@[A-Za-z0-9.-]+\.[A-Za-z]+$'::text)),
    CONSTRAINT user_first_name_check CHECK (((length(first_name) >= 2) AND (length(first_name) <= 50))),
    CONSTRAINT user_rating_check CHECK (((rating >= (0)::numeric) AND (rating <= (5)::numeric))),
    CONSTRAINT user_role_check CHECK (((role)::text = ANY ((ARRAY['user'::character varying, 'support'::character varying, 'admin'::character varying])::text[]))),
    CONSTRAINT user_second_name_check CHECK (((length(second_name) >= 2) AND (length(second_name) <= 50)))
);


ALTER TABLE public."user" OWNER TO postgres;

--
-- Name: user_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public."user" ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.user_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: v_product_promotion; Type: VIEW; Schema: public; Owner: postgres
--

CREATE VIEW public.v_product_promotion AS
 SELECT product_id,
    bool_or(((kind = 'boost'::text) AND (expires_at > CURRENT_TIMESTAMP))) AS is_boosted,
    bool_or(((kind = 'highlight'::text) AND (expires_at > CURRENT_TIMESTAMP))) AS is_highlighted,
    max(
        CASE
            WHEN ((kind = 'boost'::text) AND (expires_at > CURRENT_TIMESTAMP)) THEN expires_at
            ELSE NULL::timestamp with time zone
        END) AS boost_expires_at,
    max(
        CASE
            WHEN ((kind = 'highlight'::text) AND (expires_at > CURRENT_TIMESTAMP)) THEN expires_at
            ELSE NULL::timestamp with time zone
        END) AS highlight_expires_at
   FROM public.promotion
  GROUP BY product_id;


ALTER VIEW public.v_product_promotion OWNER TO postgres;

--
-- Name: wallet; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.wallet (
    user_id bigint NOT NULL,
    balance bigint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT wallet_balance_check CHECK ((balance >= 0))
);


ALTER TABLE public.wallet OWNER TO postgres;

--
-- Name: wallet_transaction; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.wallet_transaction (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    amount bigint NOT NULL,
    type text NOT NULL,
    reference_id bigint,
    idempotency_key text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT wallet_transaction_type_check CHECK ((type = ANY (ARRAY['topup'::text, 'promotion_charge'::text, 'refund'::text])))
);


ALTER TABLE public.wallet_transaction OWNER TO postgres;

--
-- Name: wallet_transaction_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.wallet_transaction ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.wallet_transaction_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: support_ticket id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.support_ticket ALTER COLUMN id SET DEFAULT nextval('public.support_ticket_id_seq'::regclass);


--
-- Name: cart_item cart_item_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.cart_item
    ADD CONSTRAINT cart_item_pkey PRIMARY KEY (user_id, product_id);


--
-- Name: category_characteristic category_characteristic_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.category_characteristic
    ADD CONSTRAINT category_characteristic_pkey PRIMARY KEY (id);


--
-- Name: category category_name_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.category
    ADD CONSTRAINT category_name_key UNIQUE (name);


--
-- Name: category category_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.category
    ADD CONSTRAINT category_pkey PRIMARY KEY (id);


--
-- Name: chat chat_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chat
    ADD CONSTRAINT chat_pkey PRIMARY KEY (id);


--
-- Name: chat chat_product_id_buyer_id_seller_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chat
    ADD CONSTRAINT chat_product_id_buyer_id_seller_id_key UNIQUE (product_id, buyer_id, seller_id);


--
-- Name: favorite favorite_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.favorite
    ADD CONSTRAINT favorite_pkey PRIMARY KEY (user_id, product_id);


--
-- Name: message message_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.message
    ADD CONSTRAINT message_pkey PRIMARY KEY (id);


--
-- Name: order_item order_item_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.order_item
    ADD CONSTRAINT order_item_pkey PRIMARY KEY (order_id, product_id);


--
-- Name: order order_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."order"
    ADD CONSTRAINT order_pkey PRIMARY KEY (id);


--
-- Name: payment payment_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.payment
    ADD CONSTRAINT payment_pkey PRIMARY KEY (id);


--
-- Name: platform_setting platform_setting_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.platform_setting
    ADD CONSTRAINT platform_setting_pkey PRIMARY KEY (key);


--
-- Name: product_characteristic product_characteristic_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_characteristic
    ADD CONSTRAINT product_characteristic_pkey PRIMARY KEY (id);


--
-- Name: product_characteristic product_characteristic_product_id_category_characteristic_i_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_characteristic
    ADD CONSTRAINT product_characteristic_product_id_category_characteristic_i_key UNIQUE (product_id, category_characteristic_id);


--
-- Name: product_custom_characteristic product_custom_characteristic_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_custom_characteristic
    ADD CONSTRAINT product_custom_characteristic_pkey PRIMARY KEY (id);


--
-- Name: product_custom_characteristic product_custom_characteristic_product_id_name_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_custom_characteristic
    ADD CONSTRAINT product_custom_characteristic_product_id_name_key UNIQUE (product_id, name);


--
-- Name: product_image product_image_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_image
    ADD CONSTRAINT product_image_pkey PRIMARY KEY (id);


--
-- Name: product_image product_image_product_id_sort_order_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_image
    ADD CONSTRAINT product_image_product_id_sort_order_key UNIQUE (product_id, sort_order);


--
-- Name: product product_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product
    ADD CONSTRAINT product_pkey PRIMARY KEY (id);


--
-- Name: product_price_history product_price_history_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_price_history
    ADD CONSTRAINT product_price_history_pkey PRIMARY KEY (id);


--
-- Name: product_status_history product_status_history_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_status_history
    ADD CONSTRAINT product_status_history_pkey PRIMARY KEY (id);


--
-- Name: product_view product_view_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_view
    ADD CONSTRAINT product_view_pkey PRIMARY KEY (id);


--
-- Name: promotion promotion_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.promotion
    ADD CONSTRAINT promotion_pkey PRIMARY KEY (id);


--
-- Name: promotion_plan promotion_plan_code_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.promotion_plan
    ADD CONSTRAINT promotion_plan_code_key UNIQUE (code);


--
-- Name: promotion_plan promotion_plan_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.promotion_plan
    ADD CONSTRAINT promotion_plan_pkey PRIMARY KEY (id);


--
-- Name: refresh_token refresh_token_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.refresh_token
    ADD CONSTRAINT refresh_token_pkey PRIMARY KEY (token);


--
-- Name: review review_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.review
    ADD CONSTRAINT review_pkey PRIMARY KEY (id);


--
-- Name: review review_sender_id_product_id_receiver_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.review
    ADD CONSTRAINT review_sender_id_product_id_receiver_id_key UNIQUE (sender_id, product_id, receiver_id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: support_message support_message_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.support_message
    ADD CONSTRAINT support_message_pkey PRIMARY KEY (id);


--
-- Name: support_ticket support_ticket_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.support_ticket
    ADD CONSTRAINT support_ticket_pkey PRIMARY KEY (id);


--
-- Name: user user_email_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."user"
    ADD CONSTRAINT user_email_key UNIQUE (email);


--
-- Name: user user_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."user"
    ADD CONSTRAINT user_pkey PRIMARY KEY (id);


--
-- Name: wallet wallet_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.wallet
    ADD CONSTRAINT wallet_pkey PRIMARY KEY (user_id);


--
-- Name: wallet_transaction wallet_transaction_idempotency_key_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.wallet_transaction
    ADD CONSTRAINT wallet_transaction_idempotency_key_key UNIQUE (idempotency_key);


--
-- Name: wallet_transaction wallet_transaction_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.wallet_transaction
    ADD CONSTRAINT wallet_transaction_pkey PRIMARY KEY (id);


--
-- Name: idx_order_buyer_id_desc; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_order_buyer_id_desc ON public."order" USING btree (buyer_id, id DESC);


--
-- Name: idx_payment_status_provider_created; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_payment_status_provider_created ON public.payment USING btree (status, provider, created_at) WHERE (status = 'pending'::text);


--
-- Name: idx_product_description_trgm; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_product_description_trgm ON public.product USING gin (description public.gin_trgm_ops);


--
-- Name: idx_product_title_trgm; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_product_title_trgm ON public.product USING gin (title public.gin_trgm_ops);


--
-- Name: idx_product_view_product_viewed; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_product_view_product_viewed ON public.product_view USING btree (product_id, viewed_at);


--
-- Name: idx_promotion_product_kind_expires; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_promotion_product_kind_expires ON public.promotion USING btree (product_id, kind, expires_at DESC);


--
-- Name: idx_promotion_user_created; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_promotion_user_created ON public.promotion USING btree (user_id, created_at DESC);


--
-- Name: idx_review_receiver_id_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_review_receiver_id_id ON public.review USING btree (receiver_id, id DESC);


--
-- Name: idx_review_sender_id_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_review_sender_id_id ON public.review USING btree (sender_id, id DESC);


--
-- Name: idx_support_message_ticket_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_support_message_ticket_id ON public.support_message USING btree (ticket_id);


--
-- Name: idx_wallet_transaction_user_created; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_wallet_transaction_user_created ON public.wallet_transaction USING btree (user_id, created_at DESC);


--
-- Name: payment_provider_ref_uniq; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX payment_provider_ref_uniq ON public.payment USING btree (provider, provider_ref) WHERE (provider_ref IS NOT NULL);


--
-- Name: cart_item cart_count_trigger; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER cart_count_trigger AFTER INSERT OR DELETE ON public.cart_item FOR EACH ROW EXECUTE FUNCTION public.update_cart_count();


--
-- Name: review review_aggregates_trigger; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER review_aggregates_trigger AFTER INSERT OR DELETE OR UPDATE ON public.review FOR EACH ROW EXECUTE FUNCTION public.trg_review_aggregates();


--
-- Name: cart_item cart_item_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.cart_item
    ADD CONSTRAINT cart_item_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product(id) ON DELETE CASCADE;


--
-- Name: cart_item cart_item_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.cart_item
    ADD CONSTRAINT cart_item_user_id_fkey FOREIGN KEY (user_id) REFERENCES public."user"(id) ON DELETE CASCADE;


--
-- Name: category_characteristic category_characteristic_category_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.category_characteristic
    ADD CONSTRAINT category_characteristic_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.category(id) ON DELETE CASCADE;


--
-- Name: category category_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.category
    ADD CONSTRAINT category_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.category(id) ON DELETE SET NULL;


--
-- Name: chat chat_buyer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chat
    ADD CONSTRAINT chat_buyer_id_fkey FOREIGN KEY (buyer_id) REFERENCES public."user"(id) ON DELETE CASCADE;


--
-- Name: chat chat_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chat
    ADD CONSTRAINT chat_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product(id) ON DELETE CASCADE;


--
-- Name: chat chat_seller_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chat
    ADD CONSTRAINT chat_seller_id_fkey FOREIGN KEY (seller_id) REFERENCES public."user"(id) ON DELETE CASCADE;


--
-- Name: favorite favorite_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.favorite
    ADD CONSTRAINT favorite_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product(id) ON DELETE CASCADE;


--
-- Name: favorite favorite_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.favorite
    ADD CONSTRAINT favorite_user_id_fkey FOREIGN KEY (user_id) REFERENCES public."user"(id) ON DELETE CASCADE;


--
-- Name: message message_chat_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.message
    ADD CONSTRAINT message_chat_id_fkey FOREIGN KEY (chat_id) REFERENCES public.chat(id) ON DELETE CASCADE;


--
-- Name: message message_sender_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.message
    ADD CONSTRAINT message_sender_id_fkey FOREIGN KEY (sender_id) REFERENCES public."user"(id) ON DELETE CASCADE;


--
-- Name: order order_buyer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."order"
    ADD CONSTRAINT order_buyer_id_fkey FOREIGN KEY (buyer_id) REFERENCES public."user"(id) ON DELETE RESTRICT;


--
-- Name: order order_chat_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."order"
    ADD CONSTRAINT order_chat_id_fkey FOREIGN KEY (chat_id) REFERENCES public.chat(id) ON DELETE SET NULL;


--
-- Name: order_item order_item_order_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.order_item
    ADD CONSTRAINT order_item_order_id_fkey FOREIGN KEY (order_id) REFERENCES public."order"(id) ON DELETE CASCADE;


--
-- Name: order_item order_item_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.order_item
    ADD CONSTRAINT order_item_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product(id) ON DELETE RESTRICT;


--
-- Name: payment payment_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.payment
    ADD CONSTRAINT payment_user_id_fkey FOREIGN KEY (user_id) REFERENCES public."user"(id) ON DELETE RESTRICT;


--
-- Name: platform_setting platform_setting_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.platform_setting
    ADD CONSTRAINT platform_setting_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public."user"(id);


--
-- Name: product product_category_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product
    ADD CONSTRAINT product_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.category(id) ON DELETE RESTRICT;


--
-- Name: product_characteristic product_characteristic_category_characteristic_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_characteristic
    ADD CONSTRAINT product_characteristic_category_characteristic_id_fkey FOREIGN KEY (category_characteristic_id) REFERENCES public.category_characteristic(id) ON DELETE CASCADE;


--
-- Name: product_characteristic product_characteristic_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_characteristic
    ADD CONSTRAINT product_characteristic_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product(id) ON DELETE CASCADE;


--
-- Name: product_custom_characteristic product_custom_characteristic_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_custom_characteristic
    ADD CONSTRAINT product_custom_characteristic_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product(id) ON DELETE CASCADE;


--
-- Name: product product_deleted_by_admin_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product
    ADD CONSTRAINT product_deleted_by_admin_id_fkey FOREIGN KEY (deleted_by_admin_id) REFERENCES public."user"(id);


--
-- Name: product_image product_image_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_image
    ADD CONSTRAINT product_image_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product(id) ON DELETE CASCADE;


--
-- Name: product_price_history product_price_history_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_price_history
    ADD CONSTRAINT product_price_history_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product(id) ON DELETE CASCADE;


--
-- Name: product product_seller_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product
    ADD CONSTRAINT product_seller_id_fkey FOREIGN KEY (seller_id) REFERENCES public."user"(id) ON DELETE CASCADE;


--
-- Name: product_status_history product_status_history_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_status_history
    ADD CONSTRAINT product_status_history_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product(id) ON DELETE CASCADE;


--
-- Name: product_view product_view_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_view
    ADD CONSTRAINT product_view_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product(id) ON DELETE CASCADE;


--
-- Name: product_view product_view_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.product_view
    ADD CONSTRAINT product_view_user_id_fkey FOREIGN KEY (user_id) REFERENCES public."user"(id) ON DELETE SET NULL;


--
-- Name: promotion promotion_plan_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.promotion
    ADD CONSTRAINT promotion_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.promotion_plan(id) ON DELETE RESTRICT;


--
-- Name: promotion promotion_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.promotion
    ADD CONSTRAINT promotion_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product(id) ON DELETE CASCADE;


--
-- Name: promotion promotion_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.promotion
    ADD CONSTRAINT promotion_user_id_fkey FOREIGN KEY (user_id) REFERENCES public."user"(id) ON DELETE RESTRICT;


--
-- Name: refresh_token refresh_token_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.refresh_token
    ADD CONSTRAINT refresh_token_user_id_fkey FOREIGN KEY (user_id) REFERENCES public."user"(id) ON DELETE CASCADE;


--
-- Name: review review_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.review
    ADD CONSTRAINT review_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product(id) ON DELETE CASCADE;


--
-- Name: review review_receiver_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.review
    ADD CONSTRAINT review_receiver_id_fkey FOREIGN KEY (receiver_id) REFERENCES public."user"(id) ON DELETE CASCADE;


--
-- Name: review review_sender_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.review
    ADD CONSTRAINT review_sender_id_fkey FOREIGN KEY (sender_id) REFERENCES public."user"(id) ON DELETE CASCADE;


--
-- Name: support_message support_message_ticket_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.support_message
    ADD CONSTRAINT support_message_ticket_id_fkey FOREIGN KEY (ticket_id) REFERENCES public.support_ticket(id) ON DELETE CASCADE;


--
-- Name: support_message support_message_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.support_message
    ADD CONSTRAINT support_message_user_id_fkey FOREIGN KEY (user_id) REFERENCES public."user"(id);


--
-- Name: support_ticket support_ticket_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.support_ticket
    ADD CONSTRAINT support_ticket_user_id_fkey FOREIGN KEY (user_id) REFERENCES public."user"(id);


--
-- Name: wallet_transaction wallet_transaction_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.wallet_transaction
    ADD CONSTRAINT wallet_transaction_user_id_fkey FOREIGN KEY (user_id) REFERENCES public."user"(id) ON DELETE RESTRICT;


--
-- Name: wallet wallet_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.wallet
    ADD CONSTRAINT wallet_user_id_fkey FOREIGN KEY (user_id) REFERENCES public."user"(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict iuvlNFybJO1pdAjHhfh5LXc1kbqSe0aaLO5W2NCNTY2pevMBiTADibYJeH7cIAZ

