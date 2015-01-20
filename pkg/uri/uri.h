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

#ifndef __URI_H__
#define __URI_H__


/**
 * MAX_URI_LENGTH:
 *
 * The definition of the URI regexp in the above RFC has no size limit
 * In practice they are usually relativey short except for the
 * data URI scheme as defined in RFC 2397. Even for data URI the usual
 * maximum size before hitting random practical limits is around 64 KB
 * and 4KB is usually a maximum admitted limit for proper operations.
 * The value below is more a security limit than anything else and
 * really should never be hit by 'normal' operations
 * Set to 1 MByte in 2012, this is only enforced on output
 */
#define MAX_URI_LENGTH (1024 * 1024)

/*
 * Old rule from 2396 used in legacy handling code
 * alpha    = lowalpha | upalpha
 */
#define IS_ALPHA(x) (IS_LOWALPHA(x) || IS_UPALPHA(x))

/*
 * lowalpha = "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" |
 *            "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" |
 *            "u" | "v" | "w" | "x" | "y" | "z"
 */

#define IS_LOWALPHA(x) (((x) >= 'a') && ((x) <= 'z'))

/*
 * upalpha = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" |
 *           "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" |
 *           "U" | "V" | "W" | "X" | "Y" | "Z"
 */
#define IS_UPALPHA(x) (((x) >= 'A') && ((x) <= 'Z'))

#ifdef IS_DIGIT
#undef IS_DIGIT
#endif
/*
 * digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9"
 */
#define IS_DIGIT(x) (((x) >= '0') && ((x) <= '9'))

/*
 * alphanum = alpha | digit
 */

#define IS_ALPHANUM(x) (IS_ALPHA(x) || IS_DIGIT(x))

/*
 * mark = "-" | "_" | "." | "!" | "~" | "*" | "'" | "(" | ")"
 */

#define IS_MARK(x) (((x) == '-') || ((x) == '_') || ((x) == '.') ||     \
                    ((x) == '!') || ((x) == '~') || ((x) == '*') ||     \
                    ((x) == '\'') || ((x) == '(') || ((x) == ')'))

/*
 * unwise = "{" | "}" | "|" | "\" | "^" | "`"
 */

#define IS_UNWISE(p)                                                    \
        (((*(p) == '{')) || ((*(p) == '}')) || ((*(p) == '|')) ||       \
         ((*(p) == '\\')) || ((*(p) == '^')) || ((*(p) == '[')) ||      \
         ((*(p) == ']')) || ((*(p) == '`')))
/*
 * reserved = ";" | "/" | "?" | ":" | "@" | "&" | "=" | "+" | "$" | "," |
 *            "[" | "]"
 */

#define IS_RESERVED(x) (((x) == ';') || ((x) == '/') || ((x) == '?') || \
                        ((x) == ':') || ((x) == '@') || ((x) == '&') || \
                        ((x) == '=') || ((x) == '+') || ((x) == '$') || \
                        ((x) == ',') || ((x) == '[') || ((x) == ']'))
/*
 * unreserved = alphanum | mark
 */

#define IS_UNRESERVED(x) (IS_ALPHANUM(x) || IS_MARK(x))

/*
 * Skip to next pointer char, handle escaped sequences
 */

#define NEXT(p) ((*p == '%')? p += 3 : p++)

/*
 * Productions from the spec.
 *
 *    authority     = server | reg_name
 *    reg_name      = 1*( unreserved | escaped | "$" | "," |
 *                        ";" | ":" | "@" | "&" | "=" | "+" )
 *
 * path          = [ abs_path | opaque_part ]
 */


/************************************************************************
 *                                                                      *
 *                         RFC 3986 parser                              *
 *                                                                      *
 ************************************************************************/

#define ISA_DIGIT(p) ((*(p) >= '0') && (*(p) <= '9'))
#define ISA_ALPHA(p) (((*(p) >= 'a') && (*(p) <= 'z')) ||       \
                      ((*(p) >= 'A') && (*(p) <= 'Z')))
#define ISA_HEXDIG(p)                                           \
        (ISA_DIGIT(p) || ((*(p) >= 'a') && (*(p) <= 'f')) ||    \
         ((*(p) >= 'A') && (*(p) <= 'F')))

/*
 *    sub-delims    = "!" / "$" / "&" / "'" / "(" / ")"
 *                     / "*" / "+" / "," / ";" / "="
 */
#define ISA_SUB_DELIM(p)                                                \
        (((*(p) == '!')) || ((*(p) == '$')) || ((*(p) == '&')) ||       \
         ((*(p) == '(')) || ((*(p) == ')')) || ((*(p) == '*')) ||       \
         ((*(p) == '+')) || ((*(p) == ',')) || ((*(p) == ';')) ||       \
         ((*(p) == '=')) || ((*(p) == '\'')))

/*
 *    gen-delims    = ":" / "/" / "?" / "#" / "[" / "]" / "@"
 */
#define ISA_GEN_DELIM(p)                                                \
        (((*(p) == ':')) || ((*(p) == '/')) || ((*(p) == '?')) ||       \
         ((*(p) == '#')) || ((*(p) == '[')) || ((*(p) == ']')) ||       \
         ((*(p) == '@')))

/*
 *    reserved      = gen-delims / sub-delims
 */
#define ISA_RESERVED(p) (ISA_GEN_DELIM(p) || (ISA_SUB_DELIM(p)))

/*
 *    unreserved    = ALPHA / DIGIT / "-" / "." / "_" / "~"
 */
#define ISA_UNRESERVED(p)                                       \
        ((ISA_ALPHA(p)) || (ISA_DIGIT(p)) || ((*(p) == '-')) || \
         ((*(p) == '.')) || ((*(p) == '_')) || ((*(p) == '~')))

/*
 *    pct-encoded   = "%" HEXDIG HEXDIG
 */
#define ISA_PCT_ENCODED(p)                                              \
        ((*(p) == '%') && (ISA_HEXDIG(p + 1)) && (ISA_HEXDIG(p + 2)))

/*
 *    pchar         = unreserved / pct-encoded / sub-delims / ":" / "@"
 */
#define ISA_PCHAR(p)                            \
        (ISA_UNRESERVED(p) ||                   \
         ISA_PCT_ENCODED(p) ||                  \
         ISA_SUB_DELIM(p) ||                    \
         ((*(p) == ':')) || ((*(p) == '@')))

#define URI_TRIM(uri)                           \
        if (uri != NULL) {                      \
                if (uri->scheme != NULL)        \
                        FREE(uri->scheme);      \
                if (uri->server != NULL)        \
                        FREE(uri->server);      \
                if (uri->user != NULL)          \
                        FREE(uri->user);        \
                if (uri->path != NULL)          \
                        FREE(uri->path);        \
                if (uri->fragment != NULL)      \
                        FREE(uri->fragment);    \
                if (uri->opaque != NULL)        \
                        FREE(uri->opaque);      \
                if (uri->authority != NULL)     \
                        FREE(uri->authority);   \
                if (uri->query != NULL)         \
                        FREE(uri->query);       \
        }

#define FREE(ptr)                               \
        if (ptr != NULL) {                      \
        free ((void *)ptr);                     \
        ptr = (void *)0xeeeeeeee;               \
        }

#define URI_FREE(uri)                           \
        if (uri != NULL) {                      \
                URI_TRIM(uri);                  \
                FREE(uri);                      \
        }


#define is_hex(c)                               \
        (((c >= '0') && (c <= '9')) ||          \
         ((c >= 'a') && (c <= 'f')) ||          \
         ((c >= 'A') && (c <= 'F')))

/**
 * bURI:
 *
 * A parsed bURI reference. This is a struct containing the various fields
 * as described in RFC 2396 but separated for further processing.
 */

typedef struct _bURI bURI;
typedef bURI *bURIptr;
struct _bURI {
        char *scheme;       /* the bURI scheme */
        char *opaque;       /* opaque part */
        char *authority;    /* the authority part */
        char *server;       /* the server part */
        char *user;         /* the user part */
        int port;           /* the port number */
        char *path;         /* the path string */
        char *fragment;     /* the fragment identifier */
        int  cleanup;       /* parsing potentially unclean bURI */
        char *query;        /* the query string (as it appears in the bURI) */
        char *query_raw;    /* the query string (as it appears in the bURI) */
};

/************************************************************************
 *                                                                      *
 *                      Generic URI structure functions                 *
 *                                                                      *
 ************************************************************************/

int minio_uri_parse_into(bURI *, const char *);
char *minio_uri_resolve_relative (const char *, const char *);
int minio_uri_parse(const char *, bURI **);
bURI *minio_uri_parse_raw(const char *, int);
bURI *minio_uri_new(void);
char *minio_uri_to_string(bURI *);
char *minio_uri_string_unescape(const char *, int, char *);
char *minio_uri_string_escape(const char *, const char *);

#endif /* __URI_H__ */
