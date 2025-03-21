// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalFile_Stmts(t *testing.T) {
	path := filepath.Join("testdata", "lex")
	dir, err := NewLocalDir(path)
	require.NoError(t, err)
	files, err := dir.Files()
	require.NoError(t, err)
	for _, f := range files {
		sc := &Scanner{
			ScannerOptions: ScannerOptions{
				MatchBegin:       true,
				MatchBeginAtomic: true,
				MatchDollarQuote: true,
				BackslashEscapes: true,
				EscapedStringExt: true,
				HashComments:     !strings.Contains(f.Name(), "_pg"),
				GoCommand:        strings.Contains(f.Name(), "_ms"),
			},
		}
		decls, err := sc.Scan(string(f.Bytes()))
		require.NoErrorf(t, err, "file: %s", f.Name())
		buf, err := os.ReadFile(filepath.Join(path, f.Name()+".golden"))
		require.NoError(t, err)
		stmts := make([]string, len(decls))
		for i, s := range decls {
			stmts[i] = s.Text
		}
		require.Equalf(t, string(buf), strings.Join(stmts, "\n-- end --\n"), "mismatched statements in file %q", f.Name())
	}
}

func TestScanner_StmtsGroup(t *testing.T) {
	scan := &Scanner{}
	scan.MatchBegin = true
	path := filepath.Join("testdata", "lexgroup")
	dir, err := NewLocalDir(path)
	require.NoError(t, err)
	files, err := dir.Files()
	require.NoError(t, err)
	for _, f := range files {
		stmts, err := scan.Scan(string(f.Bytes()))
		require.NoErrorf(t, err, "file: %s", f.Name())
		buf, err := os.ReadFile(filepath.Join(path, f.Name()+".golden"))
		require.NoError(t, err)
		got := make([]string, len(stmts))
		for i, s := range stmts {
			got[i] = s.Text
		}
		require.Equalf(t, string(buf), strings.Join(got, "\n-- end --\n"), "mismatched statements in file %q", f.Name())
	}
}

func TestScanner_EscapedStrings(t *testing.T) {
	path := filepath.Join("testdata", "lexescaped")
	dir, err := NewLocalDir(path)
	require.NoError(t, err)
	files, err := dir.Files()
	require.NoError(t, err)
	require.Len(t, files, 2, "tests should be updated")
	scan := &Scanner{}
	scan.BackslashEscapes = true
	stmts, err := scan.Scan(string(files[0].Bytes()))
	require.NoError(t, err)
	buf, err := os.ReadFile(filepath.Join(path, files[0].Name()+".golden"))
	require.NoError(t, err)
	got := make([]string, len(stmts))
	for i, s := range stmts {
		got[i] = s.Text
	}
	require.Equalf(t, string(buf), strings.Join(got, "\n-- end --\n"), "mismatched statements in file %q", files[0].Name())
	_, err = scan.Scan(string(files[1].Bytes()))
	require.EqualError(t, err, `4:40: unclosed quote '\''`, "escaped strings conflicts with standard strings")

	scan.BackslashEscapes = false
	scan.EscapedStringExt = true
	stmts, err = scan.Scan(string(files[0].Bytes()))
	require.EqualError(t, err, `4:42: unclosed quote '\''`, "disabled escaped strings should fail parse of escaped strings without the extension")
	stmts, err = scan.Scan(string(files[1].Bytes()))
	require.NoError(t, err)
	buf, err = os.ReadFile(filepath.Join(path, files[1].Name()+".golden"))
	require.NoError(t, err)
	got = make([]string, len(stmts))
	for i, s := range stmts {
		got[i] = s.Text
	}
	require.Equalf(t, string(buf), strings.Join(got, "\n-- end --\n"), "mismatched statements in file %q", files[1].Name())
}

func TestScanner_BeginTryCatch(t *testing.T) {
	path := filepath.Join("testdata", "lexbegintry")
	dir, err := NewLocalDir(path)
	require.NoError(t, err)
	files, err := dir.Files()
	require.NoError(t, err)
	for _, f := range files {
		sc := &Scanner{
			ScannerOptions: ScannerOptions{
				MatchBegin:         true,
				MatchBeginAtomic:   true,
				MatchBeginTryCatch: true,
				MatchDollarQuote:   true,
				BackslashEscapes:   true,
				EscapedStringExt:   true,
				HashComments:       false,
			},
		}
		decls, err := sc.Scan(string(f.Bytes()))
		require.NoErrorf(t, err, "file: %s", f.Name())
		buf, err := os.ReadFile(filepath.Join(path, f.Name()+".golden"))
		require.NoError(t, err)
		stmts := make([]string, len(decls))
		for i, s := range decls {
			stmts[i] = s.Text
		}
		require.Equalf(t, string(buf), strings.Join(stmts, "\n-- end --\n"), "mismatched statements in file %q", f.Name())
	}
}

func TestScanner_SQLServer(t *testing.T) {
	scan := &Scanner{}
	scan.MatchBegin = true
	scan.BeginEndTerminator = true
	path := filepath.Join("testdata", "sqlserver")
	dir, err := NewLocalDir(path)
	require.NoError(t, err)
	files, err := dir.Files()
	require.NoError(t, err)
	for _, f := range files {
		stmts, err := scan.Scan(string(f.Bytes()))
		require.NoErrorf(t, err, "file: %s", f.Name())
		buf, err := os.ReadFile(filepath.Join(path, f.Name()+".golden"))
		require.NoError(t, err)
		got := make([]string, len(stmts))
		for i, s := range stmts {
			got[i] = s.Text
		}
		require.Equalf(t, string(buf), strings.Join(got, "\n-- end --\n"), "mismatched statements in file %q", f.Name())
	}
}

func TestLocalFile_StmtDecls(t *testing.T) {
	f := `cmd0;
-- test
cmd1;

-- hello
-- world
cmd2;

-- skip
-- this
# comment

/* Skip this as well */

# Skip this
/* one */

# command
cmd3;

/* comment1 */
/* comment2 */
cmd4;

--atlas:nolint
-- atlas:nolint destructive
cmd5;

#atlas:lint error
/*atlas:nolint DS101*/
/* atlas:lint not a directive */
/*
atlas:lint not a directive
*/
cmd6;

-- atlas:nolint
cmd7;
`
	sc := &Scanner{
		ScannerOptions: ScannerOptions{
			MatchBegin:       true,
			MatchBeginAtomic: true,
			MatchDollarQuote: true,
			BackslashEscapes: true,
			EscapedStringExt: true,
			HashComments:     true,
		},
	}
	stmts, err := sc.Scan(f)
	require.NoError(t, err)
	require.Len(t, stmts, 8)

	require.Equal(t, "cmd0;", stmts[0].Text)
	require.Equal(t, 0, stmts[0].Pos, "start of the file")

	require.Equal(t, "cmd1;", stmts[1].Text)
	require.Equal(t, strings.Index(f, "cmd1;"), stmts[1].Pos)
	require.Equal(t, []string{"-- test\n"}, stmts[1].Comments)

	require.Equal(t, "cmd2;", stmts[2].Text)
	require.Equal(t, strings.Index(f, "cmd2;"), stmts[2].Pos)
	require.Equal(t, []string{"-- hello\n", "-- world\n"}, stmts[2].Comments)

	require.Equal(t, "cmd3;", stmts[3].Text)
	require.Equal(t, strings.Index(f, "cmd3;"), stmts[3].Pos)
	require.Equal(t, []string{"# command\n"}, stmts[3].Comments)

	require.Equal(t, "cmd4;", stmts[4].Text)
	require.Equal(t, strings.Index(f, "cmd4;"), stmts[4].Pos)
	require.Equal(t, []string{"/* comment1 */", "/* comment2 */"}, stmts[4].Comments)

	require.Equal(t, "cmd5;", stmts[5].Text)
	require.Equal(t, strings.Index(f, "cmd5;"), stmts[5].Pos)
	require.Equal(t, []string{"--atlas:nolint\n", "-- atlas:nolint destructive\n"}, stmts[5].Comments)
	require.Equal(t, []string{"", "destructive"}, stmts[5].Directive("nolint"))

	require.Equal(t, "cmd6;", stmts[6].Text)
	require.Equal(t, strings.Index(f, "cmd6;"), stmts[6].Pos)
	require.Equal(t, []string{"#atlas:lint error\n", "/*atlas:nolint DS101*/", "/* atlas:lint not a directive */", "/*\natlas:lint not a directive\n*/"}, stmts[6].Comments)
	require.Equal(t, []string{"error"}, stmts[6].Directive("lint"))
	require.Equal(t, []string{"DS101"}, stmts[6].Directive("nolint"))

	require.Equal(t, "cmd7;", stmts[7].Text)
	require.Equal(t, []string{""}, stmts[7].Directive("nolint"))
}

func TestLex_Errors(t *testing.T) {
	for _, tt := range []struct {
		name, stmt, err string
	}{
		{
			name: "unclosed single at 1:1",
			stmt: "'this quote is unclosed at 1:1",
			err:  "1:1: unclosed quote '\\''",
		},
		{
			name: "unclosed single at 1:6",
			stmt: "12345'this quote is unclosed at pos 7",
			err:  "1:6: unclosed quote '\\''",
		},
		{
			name: "unclosed single at EOS",
			stmt: "unclosed '",
			err:  "1:10: unclosed quote '\\''",
		},
		{
			name: "unclosed double at 1:1",
			stmt: "\"unclosed double",
			err:  "1:1: unclosed quote '\"'",
		},
		{
			name: "unclosed double at 2:2",
			stmt: "unclosed double at 2:2\n \"",
			err:  "2:2: unclosed quote '\"'",
		},
		{
			name: "unclosed double at 5:5",
			stmt: "unclosed double at 2:2\n\n\n\n1234\"",
			err:  "5:5: unclosed quote '\"'",
		},
		{
			name: "unclosed parentheses at 1:1",
			stmt: "(unclosed parentheses",
			err:  "1:1: unclosed '('",
		},
		{
			name: "unclosed parentheses at 1:3",
			stmt: "()(unclosed parentheses",
			err:  "1:3: unclosed '('",
		},
		{
			name: "unexpected parentheses at 1:5",
			stmt: "1234)6789",
			err:  "1:5: unexpected ')'",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Stmts(tt.stmt)
			require.EqualError(t, err, tt.err)
		})
	}
}
