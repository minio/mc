/**
 * Summary: library of generic URI related routines
 * Description: library of generic URI related routines
 *              Implements RFC 2396
 *
 *  Copyright (C) 1998-2012 Daniel Veillard.  All Rights Reserved.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.  IN NO EVENT SHALL THE
 * DANIEL VEILLARD BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
 * IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 *
 * Except as contained in this notice, the name of Daniel Veillard shall not
 * be used in advertising or otherwise to promote the sale, use or other
 * dealings in this Software without prior written authorization from him.
 *
 * daniel@veillard.com
 *
 * Mini Object Storage, (C) 2014,2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include <string.h>
#include <stdio.h>
#include <stdlib.h>

#include "uri.h"

/**
 * minio_rfc3986_scheme:
 * @uri:  pointer to an URI structure
 * @str:  pointer to the string to analyze
 *
 * Parse an URI scheme
 *
 * ALPHA *( ALPHA / DIGIT / "+" / "-" / "." )
 *
 * Returns 0 or the error code
 */

static int
minio_rfc3986_scheme(bURI *uri, const char **str)
{
        const char *cur = NULL;

        if (str == NULL)
                return -1;
        cur = *str;
        if (!ISA_ALPHA(cur))
                return 2;
        cur++;
        while (ISA_ALPHA(cur) || ISA_DIGIT(cur) ||
               (*cur == '+') || (*cur == '-') || (*cur == '.'))
                cur++;
        if (uri != NULL) {
                if (uri->scheme != NULL)
                        free(uri->scheme);
                uri->scheme = strndup(*str, cur - *str);
        }
        *str = cur;
        return 0;
}

/**
 * minio_rfc3986_fragment:
 * @uri:  pointer to an URI structure
 * @str:  pointer to the string to analyze
 *
 * Parse the query part of an URI
 *
 * fragment      = *( pchar / "/" / "?" )
 * NOTE: the strict syntax as defined by 3986 does not allow '[' and ']'
 *       in the fragment identifier but this is used very broadly for
 *       xpointer scheme selection, so we are allowing it here to not break
 *       for example all the DocBook processing chains.
 *
 * Returns 0 or the error code
 */
static int
minio_rfc3986_fragment(bURI *uri, const char **str)
{
        const char *cur = NULL;

        if (str == NULL)
                return -1;

        cur = *str;

        while ((ISA_PCHAR(cur)) || (*cur == '/') || (*cur == '?') ||
               (*cur == '[') || (*cur == ']') ||
               ((uri != NULL) && (uri->cleanup & 1) && (IS_UNWISE(cur))))
                NEXT(cur);
        if (uri != NULL) {
                if (uri->fragment != NULL)
                        free(uri->fragment);
                if (uri->cleanup & 2)
                        uri->fragment = strndup(*str, cur - *str);
                else
                        uri->fragment = minio_uri_string_unescape(*str,
                                                                      cur - *str,
                                                                      NULL);
        }

        *str = cur;
        return 0;
}

/**
 * minio_rfc3986_query:
 * @uri:  pointer to an URI structure
 * @str:  pointer to the string to analyze
 *
 * Parse the query part of an URI
 *
 * query = *uric
 *
 * Returns 0 or the error code
 */
static int
minio_rfc3986_query(bURI *uri, const char **str)
{
        const char *cur = NULL;

        if (str == NULL)
                return -1;

        cur = *str;

        while ((ISA_PCHAR(cur)) || (*cur == '/') || (*cur == '?') ||
               ((uri != NULL) && (uri->cleanup & 1) && (IS_UNWISE(cur))))
                NEXT(cur);
        if (uri != NULL) {
                if (uri->query != NULL)
                        free(uri->query);
                if (uri->cleanup & 2)
                        uri->query = strndup(*str, cur - *str);
                else
                        uri->query = minio_uri_string_unescape(*str, cur - *str,
                                                                   NULL);
                /* Save the raw bytes of the query as well.
                 * See: http://mail.gnome.org/archives/xml/2007-April/thread.html#00114
                 */
                if (uri->query_raw != NULL)
                        free(uri->query_raw);
                uri->query_raw = strndup (*str, cur - *str);
        }
        *str = cur;
        return 0;
}

/**
 * minio_rfc3986_port:
 * @uri:  pointer to an URI structure
 * @str:  the string to analyze
 *
 * Parse a port  part and fills in the appropriate fields
 * of the @uri structure
 *
 * port          = *DIGIT
 *
 * Returns 0 or the error code
 */
static int
minio_rfc3986_port(bURI *uri, const char **str)
{
        const char *cur = NULL;
        if (str == NULL)
                return -1;

        cur = *str;

        if (ISA_DIGIT(cur)) {
                if (uri != NULL)
                        uri->port = 0;
                while (ISA_DIGIT(cur)) {
                        if (uri != NULL)
                                uri->port = uri->port * 10 + (*cur - '0');
                        cur++;
                }
                *str = cur;
                return 0;
        }
        return 1;
}

/**
 * minio_rfc3986_user_info:
 * @uri:  pointer to an URI structure
 * @str:  the string to analyze
 *
 * Parse an user informations part and fills in the appropriate fields
 * of the @uri structure
 *
 * userinfo      = *( unreserved / pct-encoded / sub-delims / ":" )
 *
 * Returns 0 or the error code
 */

static int
minio_rfc3986_user_info(bURI *uri, const char **str)
{
        const char *cur = NULL;

        if (str == NULL)
                return -1;

        cur = *str;
        while (ISA_UNRESERVED(cur) || ISA_PCT_ENCODED(cur) ||
               ISA_SUB_DELIM(cur) || (*cur == ':'))
                NEXT(cur);
        if (*cur == '@') {
                if (uri != NULL) {
                        if (uri->user != NULL) free(uri->user);
                        if (uri->cleanup & 2)
                                uri->user = strndup(*str, cur - *str);
                        else
                                uri->user = minio_uri_string_unescape(*str,
                                                                          cur - *str,
                                                                          NULL);
                }
                *str = cur;
                return 0;
        }
        return 1;
}

/**
 * minio_rfc3986_dec_octet:
 * @str:  the string to analyze
 *
 *    dec-octet     = DIGIT                 ; 0-9
 *                  / %x31-39 DIGIT         ; 10-99
 *                  / "1" 2DIGIT            ; 100-199
 *                  / "2" %x30-34 DIGIT     ; 200-249
 *                  / "25" %x30-35          ; 250-255
 *
 * Skip a dec-octet.
 *
 * Returns 0 if found and skipped, 1 otherwise
 */
static int
minio_rfc3986_dec_octet(const char **str)
{
        const char *cur = NULL;
        if (str == NULL)
                return -1;
        cur = *str;

        if (!(ISA_DIGIT(cur)))
                return 1;
        if (!ISA_DIGIT(cur+1))
                cur++;
        else if ((*cur != '0') && (ISA_DIGIT(cur + 1)) && (!ISA_DIGIT(cur+2)))
                cur += 2;
        else if ((*cur == '1') && (ISA_DIGIT(cur + 1)) && (ISA_DIGIT(cur + 2)))
                cur += 3;
        else if ((*cur == '2') && (*(cur + 1) >= '0') &&
                 (*(cur + 1) <= '4') && (ISA_DIGIT(cur + 2)))
                cur += 3;
        else if ((*cur == '2') && (*(cur + 1) == '5') &&
                 (*(cur + 2) >= '0') && (*(cur + 1) <= '5'))
                cur += 3;
        else
                return 1;
        *str = cur;
        return 0;
}
/**
 * minio_rfc3986_host:
 * @uri:  pointer to an URI structure
 * @str:  the string to analyze
 *
 * Parse an host part and fills in the appropriate fields
 * of the @uri structure
 *
 * host          = IP-literal / IPv4address / reg-name
 * IP-literal    = "[" ( IPv6address / IPvFuture  ) "]"
 * IPv4address   = dec-octet "." dec-octet "." dec-octet "." dec-octet
 * reg-name      = *( unreserved / pct-encoded / sub-delims )
 *
 * Returns 0 or the error code
 */
static int
minio_rfc3986_host(bURI *uri, const char **str)
{
        const char *cur  = NULL;
        const char *host = NULL;

        if (str == NULL)
                return -1;

        cur = *str;
        if (cur == NULL)
                return -1;

        host = cur;
        /*
         * IPv6 and future addressing scheme are enclosed between brackets
         */
        if (*cur == '[') {
                cur++;
                while ((*cur != ']') && (*cur != 0))
                        cur++;
                if (*cur != ']')
                        return 1;
                cur++;
                goto found;
        }
        /*
         * try to parse an IPv4
         */
        if (ISA_DIGIT(cur)) {
                if (minio_rfc3986_dec_octet(&cur) != 0)
                        goto notipv4;
                if (*cur != '.')
                        goto notipv4;
                cur++;
                if (minio_rfc3986_dec_octet(&cur) != 0)
                        goto notipv4;
                if (*cur != '.')
                        goto notipv4;
                if (minio_rfc3986_dec_octet(&cur) != 0)
                        goto notipv4;
                if (*cur != '.')
                        goto notipv4;
                if (minio_rfc3986_dec_octet(&cur) != 0)
                        goto notipv4;
                goto found;
        notipv4:
                cur = *str;
        }
        /*
         * then this should be a hostname which can be empty
         */
        while (ISA_UNRESERVED(cur) || ISA_PCT_ENCODED(cur)
               || ISA_SUB_DELIM(cur))
                NEXT(cur);
found:
        if (uri != NULL) {
                if (uri->authority != NULL)
                        free(uri->authority);
                uri->authority = NULL;
                if (uri->server != NULL)
                        free(uri->server);
                if (cur != host) {
                        if (uri->cleanup & 2)
                                uri->server = strndup(host, cur - host);
                        else
                                uri->server = minio_uri_string_unescape(host,
                                                                            cur - host,
                                                                            NULL);
                } else
                        uri->server = NULL;
        }
        *str = cur;
        return 0;
}

/**
 * minio_rfc3986_authority:
 * @uri:  pointer to an URI structure
 * @str:  the string to analyze
 *
 * Parse an authority part and fills in the appropriate fields
 * of the @uri structure
 *
 * authority     = [ userinfo "@" ] host [ ":" port ]
 *
 * Returns 0 or the error code
 */
static int
minio_rfc3986_authority(bURI *uri, const char **str)
{
        const char *cur = NULL;
        int ret         = 0;

        if (str == NULL)
                return -1;

        cur = *str;

        ret = minio_rfc3986_user_info(uri, &cur);

        if ((ret != 0) || (*cur != '@'))
                cur = *str;
        else
                cur++;
        if (minio_rfc3986_host(uri, &cur) != 0)
                return 1;
        if (*cur == ':') {
                cur++;
                if (minio_rfc3986_port(uri, &cur) != 0)
                        return 1;
        }
        *str = cur;
        return 0;
}

/**
 * minio_rfc3986_segment:
 * @str:  the string to analyze
 * @forbid: an optional forbidden character
 * @empty: allow an empty segment
 *
 * Parse a segment and fills in the appropriate fields
 * of the @uri structure
 *
 * segment       = *pchar
 * segment-nz    = 1*pchar
 * segment-nz-nc = 1*( unreserved / pct-encoded / sub-delims / "@" )
 *               ; non-zero-length segment without any colon ":"
 *
 * Returns 0 or the error code
 */
static int
minio_rfc3986_segment(const char **str, char forbid, int empty)
{
        const char *cur = NULL;
        int ret         = 0;

        if (str == NULL) {
                ret = -1;
                goto out;
        }
        cur = *str;
        if (!ISA_PCHAR(cur)) {
                if (!empty)
                        ret = 1;
                goto out;
        }
        while (ISA_PCHAR(cur) && (*cur != forbid))
                NEXT(cur);
        *str = cur;
out:
        return ret;
}

/**
 * minio_rfc3986_path_ab_empty:
 * @uri:  pointer to an URI structure
 * @str:  the string to analyze
 *
 * Parse an path absolute or empty and fills in the appropriate fields
 * of the @uri structure
 *
 * path-abempty  = *( "/" segment )
 *
 * Returns 0 or the error code
 */
static int
minio_rfc3986_path_ab_empty(bURI *uri, const char **str)
{
        const char *cur = NULL;
        int ret         = 0;

        if (str == NULL) {
                ret = -1;
                goto out;
        }

        cur = *str;

        while (*cur == '/') {
                cur++;
                ret = minio_rfc3986_segment(&cur, 0, 1);
                if (ret != 0)
                        goto out;
        }
        if (uri != NULL) {
                if (uri->path != NULL)
                        free(uri->path);
                if (*str != cur) {
                        if (uri->cleanup & 2)
                                uri->path = strndup(*str, cur - *str);
                        else
                                uri->path = minio_uri_string_unescape(*str,
                                                                          cur - *str,
                                                                          NULL);
                } else {
                        uri->path = NULL;
                }
        }
        *str = cur;
out:
        return ret;
}

/**
 * minio_rfc3986_path_absolute:
 * @uri:  pointer to an URI structure
 * @str:  the string to analyze
 *
 * Parse an path absolute and fills in the appropriate fields
 * of the @uri structure
 *
 * path-absolute = "/" [ segment-nz *( "/" segment ) ]
 *
 * Returns 0 or the error code
 */
static int
minio_rfc3986_path_absolute(bURI *uri, const char **str)
{
        const char *cur = NULL;
        int ret         = 0;

        if (str == NULL) {
                ret = -1;
                goto out;
        }

        cur = *str;
        if (*cur != '/') {
                ret = 1;
                goto out;
        }

        cur++;
        ret = minio_rfc3986_segment(&cur, 0, 0);
        if (ret == 0) {
                while (*cur == '/') {
                        cur++;
                        ret = minio_rfc3986_segment(&cur, 0, 1);
                        if (ret != 0)
                                goto out;
                }
        }
        if (uri != NULL) {
                if (uri->path != NULL)
                        free(uri->path);
                if (cur != *str) {
                        if (uri->cleanup & 2)
                                uri->path = strndup(*str, cur - *str);
                        else
                                uri->path = minio_uri_string_unescape(*str,
                                                                          cur - *str,
                                                                          NULL);
                } else {
                        uri->path = NULL;
                }
        }
        *str = cur;
out:
        return ret;
}

/**
 * minio_rfc3986_path_rootless:
 * @uri:  pointer to an URI structure
 * @str:  the string to analyze
 *
 * Parse an path without root and fills in the appropriate fields
 * of the @uri structure
 *
 * path-rootless = segment-nz *( "/" segment )
 *
 * Returns 0 or the error code
 */
static int
minio_rfc3986_path_rootless(bURI *uri, const char **str)
{
        const char *cur;
        int ret;

        cur = *str;

        ret = minio_rfc3986_segment(&cur, 0, 0);
        if (ret != 0)
                return ret;
        while (*cur == '/') {
                cur++;
                ret = minio_rfc3986_segment(&cur, 0, 1);
                if (ret != 0)
                        return ret;
        }
        if (uri != NULL) {
                if (uri->path != NULL) free(uri->path);
                if (cur != *str) {
                        if (uri->cleanup & 2)
                                uri->path = strndup(*str, cur - *str);
                        else
                                uri->path = minio_uri_string_unescape(*str,
                                                                          cur - *str,
                                                                          NULL);
                } else {
                        uri->path = NULL;
                }
        }
        *str = cur;
        return 0;
}

/**
 * minio_rfc3986_path_no_scheme:
 * @uri:  pointer to an URI structure
 * @str:  the string to analyze
 *
 * Parse an path which is not a scheme and fills in the appropriate fields
 * of the @uri structure
 *
 * path-noscheme = segment-nz-nc *( "/" segment )
 *
 * Returns 0 or the error code
 */
static int
minio_rfc3986_path_no_scheme(bURI *uri, const char **str)
{
        const char *cur;
        int ret;

        cur = *str;

        ret = minio_rfc3986_segment(&cur, ':', 0);
        if (ret != 0)
                return ret;

        while (*cur == '/') {
                cur++;
                ret = minio_rfc3986_segment(&cur, 0, 1);
                if (ret != 0)
                        return ret;
        }
        if (uri != NULL) {
                if (uri->path != NULL)
                        free(uri->path);
                if (cur != *str) {
                        if (uri->cleanup & 2)
                                uri->path = strndup(*str, cur - *str);
                        else
                                uri->path = minio_uri_string_unescape(*str,
                                                                          cur - *str,
                                                                          NULL);
                } else {
                        uri->path = NULL;
                }
        }
        *str = cur;
        return 0;
}

/**
 * minio_rfc3986_hier_part:
 * @uri:  pointer to an URI structure
 * @str:  the string to analyze
 *
 * Parse an hierarchical part and fills in the appropriate fields
 * of the @uri structure
 *
 * hier-part     = "//" authority path-abempty
 *                / path-absolute
 *                / path-rootless
 *                / path-empty
 *
 * Returns 0 or the error code
 */
static int
minio_rfc3986_hier_part(bURI *uri, const char **str)
{
        const char *cur;
        int ret;

        cur = *str;

        if ((*cur == '/') && (*(cur + 1) == '/')) {
                cur += 2;
                ret = minio_rfc3986_authority(uri, &cur);
                if (ret != 0) return ret;
                ret = minio_rfc3986_path_ab_empty(uri, &cur);
                if (ret != 0) return ret;
                *str = cur;
                return 0;
        } else if (*cur == '/') {
                ret = minio_rfc3986_path_absolute(uri, &cur);
                if (ret != 0) return ret;
        } else if (ISA_PCHAR(cur)) {
                ret = minio_rfc3986_path_rootless(uri, &cur);
                if (ret != 0) return ret;
        } else {
                /* path-empty is effectively empty */
                if (uri != NULL) {
                        if (uri->path != NULL) free(uri->path);
                        uri->path = NULL;
                }
        }
        *str = cur;
        return 0;
}

/**
 * minio_rfc3986_relative_ref:
 * @uri:  pointer to an URI structure
 * @str:  the string to analyze
 *
 * Parse an URI string and fills in the appropriate fields
 * of the @uri structure
 *
 * relative-ref  = relative-part [ "?" query ] [ "#" fragment ]
 * relative-part = "//" authority path-abempty
 *               / path-absolute
 *               / path-noscheme
 *               / path-empty
 *
 * Returns 0 or the error code
 */
static int
minio_rfc3986_relative_ref(bURI *uri, const char *str)
{
        int ret;

        if ((*str == '/') && (*(str + 1) == '/')) {
                str += 2;
                ret = minio_rfc3986_authority(uri, &str);
                if (ret != 0) return(ret);
                ret = minio_rfc3986_path_ab_empty(uri, &str);
                if (ret != 0) return(ret);
        } else if (*str == '/') {
                ret = minio_rfc3986_path_absolute(uri, &str);
                if (ret != 0) return(ret);
        } else if (ISA_PCHAR(str)) {
                ret = minio_rfc3986_path_no_scheme(uri, &str);
                if (ret != 0) return(ret);
        } else {
                /* path-empty is effectively empty */
                if (uri != NULL) {
                        if (uri->path != NULL) free(uri->path);
                        uri->path = NULL;
                }
        }

        if (*str == '?') {
                str++;
                ret = minio_rfc3986_query(uri, &str);
                if (ret != 0)
                        return ret;
        }
        if (*str == '#') {
                str++;
                ret = minio_rfc3986_fragment(uri, &str);
                if (ret != 0)
                        return ret;
        }
        if (*str != 0) {
                URI_TRIM(uri);
                return 1;
        }
        return 0;
}


/**
 * minio_rfc3986:
 * @uri:  pointer to an URI structure
 * @str:  the string to analyze
 *
 * Parse an URI string and fills in the appropriate fields
 * of the @uri structure
 *
 * scheme ":" hier-part [ "?" query ] [ "#" fragment ]
 *
 * Returns 0 or the error code
 */
static int
minio_rfc3986(bURI *uri, const char *str)
{
        int ret;

        ret = minio_rfc3986_scheme(uri, &str);
        if (ret != 0)
                return ret;
        if (*str != ':')
                return 1;
        str++;

        ret = minio_rfc3986_hier_part(uri, &str);
        if (ret != 0)
                return ret;
        if (*str == '?') {
                str++;
                ret = minio_rfc3986_query(uri, &str);
                if (ret != 0)
                        return ret;
        }
        if (*str == '#') {
                str++;
                ret = minio_rfc3986_fragment(uri, &str);
                if (ret != 0)
                        return ret;
        }
        if (*str != 0) {
                URI_TRIM(uri);
                return 1;
        }
        return 0;
}

/**
 * minio_rfc3986_uri_reference:
 * @uri:  pointer to an URI structure
 * @str:  the string to analyze
 *
 * Parse an URI reference string and fills in the appropriate fields
 * of the @uri structure
 *
 * URI-reference = URI / relative-ref
 *
 * Returns 0 or the error code
 */
static int
minio_rfc3986_uri_reference(bURI *uri, const char *str)
{
        int ret;

        if (str == NULL)
                return -1;

        URI_TRIM(uri);
        /*
         * Try first to parse absolute refs, then fallback to relative if
         * it fails.
         */
        ret = minio_rfc3986(uri, str);
        if (ret != 0) {
                URI_TRIM(uri);
                ret = minio_rfc3986_relative_ref(uri, str);
                if (ret != 0) {
                        URI_TRIM(uri);
                        return ret;
                }
        }
        return ret;
}


/************************************************************************
 *                Generic URI structure functions                       *
 ************************************************************************/

/**
 * minio_uri_parse_into:
 * @uri:  pointer to an URI structure
 * @str:  the string to analyze
 *
 * Parse an URI reference string based on RFC 3986 and fills in the
 * appropriate fields of the @uri structure
 *
 * URI-reference = URI / relative-ref
 *
 * Returns 0 or the error code
 */
int
minio_uri_parse_into(bURI *uri, const char *str)
{
        return (minio_rfc3986_uri_reference(uri, str));
}

/**
 * minio_uri_parse:
 * @str:  the bURI string to analyze
 *
 * Parse an bURI based on RFC 3986
 *
 * URI-reference = [ absoluteURI | relativeURI ] [ "#" fragment ]
 *
 * Returns a newly built bURI or NULL in case of error
 */
int
minio_uri_parse(const char *str, bURI **buri)
{
        bURI *uri = NULL;
        int ret = -1;

        if (str == NULL)
                goto out;

        uri = minio_uri_new();
        if (!uri)
                goto out;

        ret = minio_uri_parse_into(uri, str);
        if (uri->scheme == (void *)0xeeeeeeee) {
                ret = -1;
                goto out;
        }

        ret = 0;
        *buri = uri;
out:
        return ret;
}

/**
 * minio_uri_parse_raw:
 * @str:  the bURI string to analyze
 * @raw:  if 1 unescaping of bURI pieces are disabled
 *
 * Parse an bURI but allows to keep intact the original fragments.
 *
 * bURI-reference = bURI / relative-ref
 *
 * Returns a newly built URI or NULL in case of error
 */
bURI *
minio_uri_parse_raw(const char *str, int raw)
{
        bURI *uri = NULL;

        if (str == NULL)
                goto out;
        uri = minio_uri_new();
        if (uri != NULL) {
                if (raw) {
                        uri->cleanup |= 2;
                }
                if (minio_uri_parse_into(uri, str)) {
                        URI_FREE(uri);
                        goto out;
                }
        }
out:
        return uri;
}

/**
 * minio_uri_new:
 *
 * Simply creates an empty URI
 *
 * Returns the new structure or NULL in case of error
 */
bURI *
minio_uri_new(void)
{
        bURI *tmp_uri;
        tmp_uri = (bURI *) malloc(sizeof(bURI));
        memset(tmp_uri, 0, sizeof(bURI));
        return tmp_uri;
}

/**
 * realloc2n:
 *
 * Function to handle properly a reallocation when saving an URI
 * Also imposes some limit on the length of an URI string output
 */
static char *
realloc2n(char *ret, int *max)
{
        char *temp;
        int tmp;

        tmp = *max * 2;
        temp = realloc(ret, (tmp + 1));
        *max = tmp;
        return temp;
}

/**
 * minio_uri_to_string:
 * @uri:  pointer to an bURI
 *
 * Save the bURI as an escaped string
 *
 * Returns a new string (to be deallocated by caller)
 */
char *
minio_uri_to_string(bURI *uri)
{
        char *ret = NULL;
        char *temp;
        const char *p;
        int len;
        int max;

        if (uri == NULL)
                return NULL;

        max = 80;
        ret = malloc(max + 1);
        len = 0;

        if (uri->scheme != NULL) {
                p = uri->scheme;
                while (*p != 0) {
                        if (len >= max) {
                                temp = realloc2n(ret, &max);
                                if (temp == NULL) goto mem_error;
                                ret = temp;
                        }
                        ret[len++] = *p++;
                }
                if (len >= max) {
                        temp = realloc2n(ret, &max);
                        if (temp == NULL) goto mem_error;
                        ret = temp;
                }
                ret[len++] = ':';
        }
        if (uri->opaque != NULL) {
                p = uri->opaque;
                while (*p != 0) {
                        if (len + 3 >= max) {
                                temp = realloc2n(ret, &max);
                                if (temp == NULL) goto mem_error;
                                ret = temp;
                        }
                        if (IS_RESERVED(*(p)) || IS_UNRESERVED(*(p)))
                                ret[len++] = *p++;
                        else {
                                int val = *(unsigned char *)p++;
                                int hi = val / 0x10, lo = val % 0x10;
                                ret[len++] = '%';
                                ret[len++] = hi + (hi > 9? 'A'-10 : '0');
                                ret[len++] = lo + (lo > 9? 'A'-10 : '0');
                        }
                }
        } else {
                if (uri->server != NULL) {
                        if (len + 3 >= max) {
                                temp = realloc2n(ret, &max);
                                if (temp == NULL) goto mem_error;
                                ret = temp;
                        }
                        ret[len++] = '/';
                        ret[len++] = '/';
                        if (uri->user != NULL) {
                                p = uri->user;
                                while (*p != 0) {
                                        if (len + 3 >= max) {
                                                temp = realloc2n(ret, &max);
                                                if (temp == NULL)
                                                        goto mem_error;
                                                ret = temp;
                                        }
                                        if ((IS_UNRESERVED(*(p))) ||
                                            ((*(p) == ';')) ||
                                            ((*(p) == ':')) ||
                                            ((*(p) == '&')) ||
                                            ((*(p) == '=')) ||
                                            ((*(p) == '+')) ||
                                            ((*(p) == '$')) ||
                                            ((*(p) == ',')))
                                                ret[len++] = *p++;
                                        else {
                                                int val = *(unsigned char *)p++;
                                                int hi = val / 0x10;
                                                int lo = val % 0x10;
                                                ret[len++] = '%';
                                                ret[len++] = hi +
                                                        (hi > 9? 'A'-10 : '0');
                                                ret[len++] = lo +
                                                        (lo > 9? 'A'-10 : '0');
                                        }
                                }
                                if (len + 3 >= max) {
                                        temp = realloc2n(ret, &max);
                                        if (temp == NULL)
                                                goto mem_error;
                                        ret = temp;
                                }
                                ret[len++] = '@';
                        }
                        p = uri->server;
                        while (*p != 0) {
                                if (len >= max) {
                                        temp = realloc2n(ret, &max);
                                        if (temp == NULL)
                                                goto mem_error;
                                        ret = temp;
                                }
                                ret[len++] = *p++;
                        }
                        if (uri->port > 0) {
                                if (len + 10 >= max) {
                                        temp = realloc2n(ret, &max);
                                        if (temp == NULL) goto mem_error;
                                        ret = temp;
                                }
                                len += snprintf(&ret[len], max - len, ":%d",
                                                uri->port);
                        }
                } else if (uri->authority != NULL) {
                        if (len + 3 >= max) {
                                temp = realloc2n(ret, &max);
                                if (temp == NULL) goto mem_error;
                                ret = temp;
                        }
                        ret[len++] = '/';
                        ret[len++] = '/';
                        p = uri->authority;
                        while (*p != 0) {
                                if (len + 3 >= max) {
                                        temp = realloc2n(ret, &max);
                                        if (temp == NULL) goto mem_error;
                                        ret = temp;
                                }
                                if ((IS_UNRESERVED(*(p))) ||
                                    ((*(p) == '$')) || ((*(p) == ',')) ||
                                    ((*(p) == ';')) ||
                                    ((*(p) == ':')) || ((*(p) == '@')) ||
                                    ((*(p) == '&')) ||
                                    ((*(p) == '=')) || ((*(p) == '+')))
                                        ret[len++] = *p++;
                                else {
                                        int val = *(unsigned char *)p++;
                                        int hi = val / 0x10, lo = val % 0x10;
                                        ret[len++] = '%';
                                        ret[len++] = hi +
                                                (hi > 9? 'A'-10 : '0');
                                        ret[len++] = lo +
                                                (lo > 9? 'A'-10 : '0');
                                }
                        }
                } else if (uri->scheme != NULL) {
                        if (len + 3 >= max) {
                                temp = realloc2n(ret, &max);
                                if (temp == NULL) goto mem_error;
                                ret = temp;
                        }
                        ret[len++] = '/';
                        ret[len++] = '/';
                }
                if (uri->path != NULL) {
                        p = uri->path;
                        /*
                         * the colon in file:///d: should not be escaped or
                         * Windows accesses fail later.
                         */
                        if ((uri->scheme != NULL) &&
                            (p[0] == '/') &&
                            (((p[1] >= 'a') && (p[1] <= 'z')) ||
                             ((p[1] >= 'A') && (p[1] <= 'Z'))) &&
                            (p[2] == ':') &&
                            (!strcmp(uri->scheme, "file"))) {
                                if (len + 3 >= max) {
                                        temp = realloc2n(ret, &max);
                                        if (temp == NULL) goto mem_error;
                                        ret = temp;
                                }
                                ret[len++] = *p++;
                                ret[len++] = *p++;
                                ret[len++] = *p++;
                        }
                        while (*p != 0) {
                                if (len + 3 >= max) {
                                        temp = realloc2n(ret, &max);
                                        if (temp == NULL) goto mem_error;
                                        ret = temp;
                                }
                                if ((IS_UNRESERVED(*(p))) || ((*(p) == '/')) ||
                                    ((*(p) == ';')) || ((*(p) == '@')) ||
                                    ((*(p) == '&')) ||
                                    ((*(p) == '=')) || ((*(p) == '+')) ||
                                    ((*(p) == '$')) ||
                                    ((*(p) == ',')))
                                        ret[len++] = *p++;
                                else {
                                        int val = *(unsigned char *)p++;
                                        int hi = val / 0x10, lo = val % 0x10;
                                        ret[len++] = '%';
                                        ret[len++] = hi +
                                                (hi > 9? 'A'-10 : '0');
                                        ret[len++] = lo +
                                                (lo > 9? 'A'-10 : '0');
                                }
                        }
                }
                if (uri->query != NULL) {
                        if (len + 1 >= max) {
                                temp = realloc2n(ret, &max);
                                if (temp == NULL) goto mem_error;
                                ret = temp;
                        }
                        ret[len++] = '?';
                        p = uri->query;
                        while (*p != 0) {
                                if (len + 1 >= max) {
                                        temp = realloc2n(ret, &max);
                                        if (temp == NULL) goto mem_error;
                                        ret = temp;
                                }
                                ret[len++] = *p++;
                        }
                }
        }
        if (uri->fragment != NULL) {
                if (len + 3 >= max) {
                        temp = realloc2n(ret, &max);
                        if (temp == NULL) goto mem_error;
                        ret = temp;
                }
                ret[len++] = '#';
                p = uri->fragment;
                while (*p != 0) {
                        if (len + 3 >= max) {
                                temp = realloc2n(ret, &max);
                                if (temp == NULL) goto mem_error;
                                ret = temp;
                        }
                        if ((IS_UNRESERVED(*(p))) || (IS_RESERVED(*(p))))
                                ret[len++] = *p++;
                        else {
                                int val = *(unsigned char *)p++;
                                int hi = val / 0x10, lo = val % 0x10;
                                ret[len++] = '%';
                                ret[len++] = hi + (hi > 9? 'A'-10 : '0');
                                ret[len++] = lo + (lo > 9? 'A'-10 : '0');
                        }
                }
        }
        if (len >= max) {
                temp = realloc2n(ret, &max);
                if (temp == NULL)
                        goto mem_error;
                ret = temp;
        }
        ret[len] = 0;
        return ret;

mem_error:
        FREE(ret);

        return NULL;
}


/**
 * minio_uri_string_unescape:
 * @str:  the string to unescape
 * @len:   the length in bytes to unescape (or <= 0 to indicate full string)
 * @target:  optional destination buffer
 *
 * Unescaping routine, but does not check that the string is an URI. The
 * output is a direct unsigned char translation of %XX values (no encoding)
 * Note that the length of the result can only be smaller or same size as
 * the input string.
 *
 * Returns a copy of the string, but unescaped, will return NULL only in case
 * of error
 */
char *
minio_uri_string_unescape(const char *str, int len, char *target)
{
        char *ret, *out;
        const char *in;

        if (str == NULL)
                return(NULL);
        if (len <= 0)
                len = strlen(str);
        if (len < 0)
                return(NULL);
        if (target == NULL) {
                ret = malloc(len + 1);
        } else
                ret = target;
        in = str;
        out = ret;
        while(len > 0) {
                if ((len > 2) && (*in == '%') &&
                    (is_hex(in[1])) && (is_hex(in[2]))) {
                        in++;
                        if ((*in >= '0') && (*in <= '9'))
                                *out = (*in - '0');
                        else if ((*in >= 'a') && (*in <= 'f'))
                                *out = (*in - 'a') + 10;
                        else if ((*in >= 'A') && (*in <= 'F'))
                                *out = (*in - 'A') + 10;
                        in++;
                        if ((*in >= '0') && (*in <= '9'))
                                *out = *out * 16 + (*in - '0');
                        else if ((*in >= 'a') && (*in <= 'f'))
                                *out = *out * 16 + (*in - 'a') + 10;
                        else if ((*in >= 'A') && (*in <= 'F'))
                                *out = *out * 16 + (*in - 'A') + 10;
                        in++;
                        len -= 3;
                        out++;
                } else {
                        *out++ = *in++;
                        len--;
                }
        }
        *out = 0;
        return ret;
}

/**
 * minio_uri_string_escape:
 * @str:  string to escape
 * @list: exception list string of chars not to escape
 *
 * This routine escapes a string to hex, ignoring reserved characters (a-z)
 * and the characters in the exception list.
 *
 * Returns a new escaped string or NULL in case of error.
 */
char *
minio_uri_string_escape(const char *str, const char *list)
{
        char *ret, ch;
        char *temp;
        const char *in;
        int len, out;

        if (str == NULL)
                return(NULL);
        if (str[0] == 0)
                return(strdup(str));
        len = strlen(str);
        if (!(len > 0)) return(NULL);

        len += 20;
        ret = malloc(len);
        in = str;
        out = 0;
        while(*in != 0) {
                if (len - out <= 3) {
                        temp = realloc2n(ret, &len);
                        ret = temp;
                }
                ch = *in;

                if ((ch != '@') && (!IS_UNRESERVED(ch)) &&
                    (!strchr(list, ch))) {
                        unsigned char val;
                        ret[out++] = '%';
                        val = ch >> 4;
                        if (val <= 9)
                                ret[out++] = '0' + val;
                        else
                                ret[out++] = 'A' + val - 0xA;
                        val = ch & 0xF;
                        if (val <= 9)
                                ret[out++] = '0' + val;
                        else
                                ret[out++] = 'A' + val - 0xA;
                        in++;
                } else {
                        ret[out++] = *in++;
                }
        }
        ret[out] = 0;
        return(ret);
}
