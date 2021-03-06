/* Copyright (C) 2013 CompleteDB LLC.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with PubSubSQL.  If not, see <http://www.gnu.org/licenses/>.
 */

package pubsubsql

import "testing"
import "fmt"

// ignores consumed tokens usifull in benchmark code
type ignoreTokenConsumer struct {
}

func (c *ignoreTokenConsumer) Consume(t *token) {
}

// prints consumed tokens on a separate line
type printlnTokenConsumer struct {
}

func (c *printlnTokenConsumer) Consume(t *token) {
	fmt.Println(t)
}

// sends consumed tokens to the channel
type chanTokenConsumer struct {
	channel chan *token
}

func (consumer *chanTokenConsumer) Consume(t *token) {
	consumer.channel <- t
	if t.typ == tokenTypeEOF {
		close(consumer.channel)
	}
}

func validateTokens(t *testing.T, expected []token, tokens chan *token) {
	breakLoop := false
	for _, e := range expected {
		g := <-tokens
		if e.typ != g.typ {
			t.Errorf("expected type " + e.typ.String() + " but got " + g.typ.String() + " value: " + g.val)
			breakLoop = true
		}
		if e.val != g.val {
			t.Errorf("expected value " + e.val + " but got " + g.val)
			breakLoop = true
		}
		if breakLoop {
			break
		}
	}
}

// BENCHMARKS

// UPDATE "go test -test.bench Update"
// update 1 column
func BenchmarkUpdate1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var consumer ignoreTokenConsumer
		lex("update table1 set col1 = 'value1' where key = '1234567890'", &consumer)
	}
}

// update 2 columns
func BenchmarkUpdate2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var consumer ignoreTokenConsumer
		lex("update table1 set col1 = 'value1', col2 = 'value2 where key = '1234567890'", &consumer)
	}
}

// update 4 columns
func BenchmarkUpdate4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var consumer ignoreTokenConsumer
		lex("update table1 set col1 = 'value1', col2 = 'value2, col3 = 'value3' col4 = value4 where key = '1234567890'", &consumer)
	}
}

// END BENCHMARKS

// STATUS
func TestStatusCommand(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex("status", &consumer)
	expected := []token{
		{tokenTypeCmdStatus, "status"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}

// STOP
func TestStopCommand(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex("stop", &consumer)
	expected := []token{
		{tokenTypeCmdStop, "stop"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}

// CLOSE
func TestCloseCommand(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex("close", &consumer)
	expected := []token{
		{tokenTypeCmdClose, "close"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}

// INSERT
func TestSqlInsertStatement(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex("insert into stocks (	ticker,bid, ask		 ) values (IBM, '34.43', 465.123)", &consumer)
	expected := []token{
		{tokenTypeSqlInsert, "insert"},
		{tokenTypeSqlInto, "into"},
		{tokenTypeSqlTable, "stocks"},
		{tokenTypeSqlLeftParenthesis, "("},
		{tokenTypeSqlColumn, "ticker"},
		{tokenTypeSqlComma, ","},
		{tokenTypeSqlColumn, "bid"},
		{tokenTypeSqlComma, ","},
		{tokenTypeSqlColumn, "ask"},
		{tokenTypeSqlRightParenthesis, ")"},
		{tokenTypeSqlValues, "values"},
		{tokenTypeSqlLeftParenthesis, "("},
		{tokenTypeSqlValue, "IBM"},
		{tokenTypeSqlComma, ","},
		{tokenTypeSqlValue, "34.43"},
		{tokenTypeSqlComma, ","},
		{tokenTypeSqlValue, "465.123"},
		{tokenTypeSqlRightParenthesis, ")"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}

// DELETE
func TestSqlDeleteStatement1(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex(" delete	 	 from stocks", &consumer)
	expected := []token{
		{tokenTypeSqlDelete, "delete"},
		{tokenTypeSqlFrom, "from"},
		{tokenTypeSqlTable, "stocks"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}

func TestSqlDeleteStatement2(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex(" delete	 	 from stocks where ticker = 'IBM'  ", &consumer)
	expected := []token{
		{tokenTypeSqlDelete, "delete"},
		{tokenTypeSqlFrom, "from"},
		{tokenTypeSqlTable, "stocks"},
		{tokenTypeSqlWhere, "where"},
		{tokenTypeSqlColumn, "ticker"},
		{tokenTypeSqlEqual, "="},
		{tokenTypeSqlValue, "IBM"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}

// SELECT
func TestSqlSelectStatement1(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex(" select * 	from stocks", &consumer)
	expected := []token{
		{tokenTypeSqlSelect, "select"},
		{tokenTypeSqlStar, "*"},
		{tokenTypeSqlFrom, "from"},
		{tokenTypeSqlTable, "stocks"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}

func TestSqlSelectStatement2(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex(" select ticker, bid, ask from stocks", &consumer)
	expected := []token{
		{tokenTypeSqlSelect, "select"},
		{tokenTypeSqlColumn, "ticker"},
		{tokenTypeSqlComma, ","},
		{tokenTypeSqlColumn, "bid"},
		{tokenTypeSqlComma, ","},
		{tokenTypeSqlColumn, "ask"},
		{tokenTypeSqlFrom, "from"},
		{tokenTypeSqlTable, "stocks"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}

func TestSqlSelectStatement3(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex(" select	* 	 from stocks where ticker = IBM", &consumer)
	expected := []token{
		{tokenTypeSqlSelect, "select"},
		{tokenTypeSqlStar, "*"},
		{tokenTypeSqlFrom, "from"},
		{tokenTypeSqlTable, "stocks"},
		{tokenTypeSqlWhere, "where"},
		{tokenTypeSqlColumn, "ticker"},
		{tokenTypeSqlEqual, "="},
		{tokenTypeSqlValue, "IBM"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}

// SUBSCRIBE
func TestSqlSubscribeStatement1(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex(" subscribe * 	from stocks", &consumer)
	expected := []token{
		{tokenTypeSqlSubscribe, "subscribe"},
		{tokenTypeSqlStar, "*"},
		{tokenTypeSqlFrom, "from"},
		{tokenTypeSqlTable, "stocks"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}

func TestSqlSubscribeStatement2(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex(" subscribe	* 	 from stocks where ticker = 'MSFT'", &consumer)
	expected := []token{
		{tokenTypeSqlSubscribe, "subscribe"},
		{tokenTypeSqlStar, "*"},
		{tokenTypeSqlFrom, "from"},
		{tokenTypeSqlTable, "stocks"},
		{tokenTypeSqlWhere, "where"},
		{tokenTypeSqlColumn, "ticker"},
		{tokenTypeSqlEqual, "="},
		{tokenTypeSqlValue, "MSFT"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}

func TestSqlSubscribeStatement3(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex(" subscribe skip	* 	 from stocks where ticker = 'MSFT'", &consumer)
	expected := []token{
		{tokenTypeSqlSubscribe, "subscribe"},
		{tokenTypeSqlSkip, "skip"},
		{tokenTypeSqlStar, "*"},
		{tokenTypeSqlFrom, "from"},
		{tokenTypeSqlTable, "stocks"},
		{tokenTypeSqlWhere, "where"},
		{tokenTypeSqlColumn, "ticker"},
		{tokenTypeSqlEqual, "="},
		{tokenTypeSqlValue, "MSFT"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}

// UNSUBSCRIBE
func TestSqlUnrsubscribeStatement1(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex("unsubscribe from stocks", &consumer)
	expected := []token{
		{tokenTypeSqlUnsubscribe, "unsubscribe"},
		{tokenTypeSqlFrom, "from"},
		{tokenTypeSqlTable, "stocks"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}

// UPDATE
func TestSqlUpdateStatement1(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex(" update stocks set bid = 140.45, ask = 142.01 ", &consumer)
	expected := []token{
		{tokenTypeSqlUpdate, "update"},
		{tokenTypeSqlTable, "stocks"},
		{tokenTypeSqlSet, "set"},
		{tokenTypeSqlColumn, "bid"},
		{tokenTypeSqlEqual, "="},
		{tokenTypeSqlValue, "140.45"},
		{tokenTypeSqlComma, ","},
		{tokenTypeSqlColumn, "ask"},
		{tokenTypeSqlEqual, "="},
		{tokenTypeSqlValue, "142.01"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}

func TestSqlUpdateStatement2(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex(" update stocks set bid = 140.45, ask = '142.01' where ticker = 'GOOG'", &consumer)
	expected := []token{
		{tokenTypeSqlUpdate, "update"},
		{tokenTypeSqlTable, "stocks"},
		{tokenTypeSqlSet, "set"},
		{tokenTypeSqlColumn, "bid"},
		{tokenTypeSqlEqual, "="},
		{tokenTypeSqlValue, "140.45"},
		{tokenTypeSqlComma, ","},
		{tokenTypeSqlColumn, "ask"},
		{tokenTypeSqlEqual, "="},
		{tokenTypeSqlValue, "142.01"},
		{tokenTypeSqlWhere, "where"},
		{tokenTypeSqlColumn, "ticker"},
		{tokenTypeSqlEqual, "="},
		{tokenTypeSqlValue, "GOOG"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}

// KEY
func TestSqlKeyStatement(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex("key stocks ticker", &consumer)
	expected := []token{
		{tokenTypeSqlKey, "key"},
		{tokenTypeSqlTable, "stocks"},
		{tokenTypeSqlColumn, "ticker"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}

// TAG
func TestSqlTagStatement(t *testing.T) {
	consumer := chanTokenConsumer{channel: make(chan *token)}
	go lex("tag stocks sector", &consumer)
	expected := []token{
		{tokenTypeSqlTag, "tag"},
		{tokenTypeSqlTable, "stocks"},
		{tokenTypeSqlColumn, "sector"},
		{tokenTypeEOF, ""}}

	validateTokens(t, expected, consumer.channel)
}
