{
  const reservedMap = {
    'ALTER': true,
    'ALL': true,
    'ADD': true,
    'AND': true,
    'AS': true,
    'ASC': true,

    'BETWEEN': true,
    'BY': true,

    'CALL': true,
    'CASE': true,
    'CREATE': true,
    'CONTAINS': true,
    'COUNT': true,
    'CURRENT_DATE': true,
    'CURRENT_TIME': true,
    'CURRENT_TIMESTAMP': true,
    'CURRENT_USER': true,

    'DELETE': true,
    'DESC': true,
    'DISTINCT': true,
    'DROP': true,

    'ELSE': true,
    'END': true,
    'EXISTS': true,
    'EXPLAIN': true,

    'FALSE': true,
    'FROM': true,
    'FULL': true,

    'GENERATED': true,
    'GROUP': true,

    'HAVING': true,

    'IN': true,
    'INNER': true,
    'INSERT': true,
    'INTO': true,
    'IS': true,

    'JOIN': true,
    // 'JSON': true,

    // 'KEY': true,

    'LEFT': true,
    'LIKE': true,
    'LIMIT': true,
    'LOW_PRIORITY': true, // for lock table

    'NOT': true,
    'NULL': true,

    'ON': true,
    'OR': true,
    'ORDER': true,
    'OUTER': true,

    'RECURSIVE': true,
    'RENAME': true,
    'READ': true, // for lock table
    'RIGHT': true,

    'SELECT': true,
    'SESSION_USER': true,
    'SET': true,
    'SHOW': true,
    'SYSTEM_USER': true,

    'TABLE': true,
    'THEN': true,
    'TRUE': true,
    'TRUNCATE': true,
    // 'TYPE': true,   // reserved (MySQL)

    'UNION': true,
    'UPDATE': true,
    'USING': true,

    'VALUES': true,

    'WITH': true,
    'WHEN': true,
    'WHERE': true,
    'WRITE': true, // for lock table

    'GLOBAL': true,
    'SESSION': true,
    'LOCAL': true,
    'PERSIST': true,
    'PERSIST_ONLY': true,
  };

  function getLocationObject() {
    return options.includeLocations ? {loc: location()} : {}
  }

  function createUnaryExpr(op, e) {
    return {
      type: 'unary_expr',
      operator: op,
      expr: e
    };
  }

  function createBinaryExpr(op, left, right) {
    return {
      type: 'binary_expr',
      operator: op,
      left: left,
      right: right
    };
  }

  function isBigInt(numberStr) {
    const previousMaxSafe = BigInt(Number.MAX_SAFE_INTEGER)
    const num = BigInt(numberStr)
    if (num < previousMaxSafe) return false
    return true
  }

  function createList(head, tail, po = 3) {
    const result = [head];
    for (let i = 0; i < tail.length; i++) {
      delete tail[i][po].tableList
      delete tail[i][po].columnList
      result.push(tail[i][po]);
    }
    return result;
  }

  function createBinaryExprChain(head, tail) {
    let result = head;
    for (let i = 0; i < tail.length; i++) {
      result = createBinaryExpr(tail[i][1], result, tail[i][3]);
    }
    return result;
  }

  function queryTableAlias(tableName) {
    const alias = tableAlias[tableName]
    if (alias) return alias
    if (tableName) return tableName
    return null
  }

  function columnListTableAlias(columnList) {
    const newColumnsList = new Set()
    const symbolChar = '::'
    for(let column of columnList.keys()) {
      const columnInfo = column.split(symbolChar)
      if (!columnInfo) {
        newColumnsList.add(column)
        break
      }
      if (columnInfo && columnInfo[1]) columnInfo[1] = queryTableAlias(columnInfo[1])
      newColumnsList.add(columnInfo.join(symbolChar))
    }
    return Array.from(newColumnsList)
  }

  function refreshColumnList(columnList) {
    const columns = columnListTableAlias(columnList)
    columnList.clear()
    columns.forEach(col => columnList.add(col))
  }

  const tableList = new Set();
  const columnList = new Set();
  const tableAlias = {};
}
start
  = __ n:(multiple_stmt) {
    return n
  }

crud_stmt
  = union_stmt
  / empty_stmt

empty_stmt
  = __ { return [] }

multiple_stmt
  = head:crud_stmt tail:(__ SEMICOLON __ crud_stmt)* {
      const headAst = head && head.ast || head
      const cur = tail && tail.length && tail[0].length >= 4 ? [headAst] : headAst;
      if (!tail) tail = []
      for (let i = 0; i < tail.length; i++) {
        if(!tail[i][3] || tail[i][3].length === 0) continue;
        cur.push(tail[i][3] && tail[i][3].ast || tail[i][3]);
      }
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: cur
      }
    }

set_op
  = KW_UNION __ s:(KW_ALL / KW_DISTINCT)? {
    return s ? `union ${s.toLowerCase()}` : 'union'
  }

select_core
  = select_stmt
  / v:value_clause {
      return { type: 'values', values: v.values }
    }

union_stmt
  = head:select_core tail:(__ set_op __ select_core)* __ ob: order_by_clause? __ l:limit_clause? {
      let cur = head
      for (let i = 0; i < tail.length; i++) {
        cur._next = tail[i][3]
        cur.set_op = tail[i][1]
        cur = cur._next
      }
      if(ob) head._orderby = ob
      if(l) head._limit = l
      return {
        tableList: Array.from(tableList),
        columnList: columnListTableAlias(columnList),
        ast: head
      }
    }
collate_expr
  = KW_COLLATE __ s:KW_ASSIGIN_EQUAL? __ ca:ident {
    return {
      type: 'collate',
      keyword: 'collate',
      collate: {
        name: ca,
        symbol: s,
      }
    }
  }
select_stmt
  = select_stmt_nake
  / s:('(' __ select_stmt __ ')') {
      return {
        ...s[2],
        parentheses_symbol: true,
      }
    }

with_clause
  = KW_WITH __ head:cte_definition tail:(__ COMMA __ cte_definition)* {
      return createList(head, tail);
    }
  / __ KW_WITH __ KW_RECURSIVE __ cte:cte_definition tail:(__ COMMA __ cte_definition)* {
      cte.recursive = true;
      return createList(cte, tail);
    }

cte_definition
  = name:(literal_string / ident_name / table_name) __ columns:cte_column_definition? __ KW_AS __ LPAREN __ stmt:union_stmt __ RPAREN {
    if (typeof name === 'string') name = { type: 'default', value: name }
    if (name.table) name = { type: 'default', value: name.table }
    return { name, stmt, columns };
  }

cte_column_definition
  = LPAREN __ l:column_ref_index __ RPAREN {
      return l
    }

select_stmt_nake
  = __ cte:with_clause? __ KW_SELECT ___
    opts:option_clause? __
    d:KW_DISTINCT?      __
    c:column_clause     __
    f:from_clause?      __
    w:where_clause?     __
    g:group_by_clause?  __
    h:having_clause?    __
    o:order_by_clause?  __
    l:limit_clause?
    fu: ('FOR'i __ KW_UPDATE)? {
      if(f) f.forEach(info => info.table && tableList.add(`select::${info.db}::${info.table}`));
      return {
          with: cte,
          type: 'select',
          options: opts,
          distinct: d,
          columns: c,
          from: f,
          where: w,
          groupby: g,
          having: h,
          orderby: o,
          limit: l,
          for_update: fu && `${fu[0]} ${fu[2][0]}`,
      };
  }

// MySQL extensions to standard SQL
option_clause
  = head:query_option tail:(__ query_option)* {
    const opts = [head];
    for (let i = 0, l = tail.length; i < l; ++i) {
      opts.push(tail[i][1]);
    }
    return opts;
  }

query_option
  = option:(
        OPT_SQL_CALC_FOUND_ROWS
        / (OPT_SQL_CACHE / OPT_SQL_NO_CACHE)
        / OPT_SQL_BIG_RESULT
        / OPT_SQL_SMALL_RESULT
        / OPT_SQL_BUFFER_RESULT
    ) { return option; }

column_clause
  = head: (KW_ALL / (STAR !ident_start) / STAR) tail:(__ COMMA __ column_list_item)* {
      columnList.add('select::null::(.*)')
      const item = {
        expr: {
          type: 'column_ref',
          table: null,
          column: '*'
        },
        as: null
      }
      if (tail && tail.length > 0) return createList(item, tail)
      return [item]
    }
  / head:column_list_item tail:(__ COMMA __ column_list_item)* {
      return createList(head, tail);
    }

column_list_item
  = tbl:(ident __ DOT)? __ STAR {
      const table = tbl && tbl[0] || null
      columnList.add(`select::${table}::(.*)`);
      return {
        expr: {
          type: 'column_ref',
          table: table,
          column: '*'
        },
        as: null
      };
    }
  / e:binary_column_expr __ alias:alias_clause? {
      if (e.type === 'double_quote_string' || e.type === 'single_quote_string') {
        columnList.add(`select::null::${e.value}`)
      }
      return { expr: e, as: alias };
    }

alias_clause
  = KW_AS ___ i:alias_ident { return i; }
  / KW_AS? __ i:ident { return i; }

from_clause
  = KW_FROM __ l:table_ref_list { return l; }

table_ref_list
  = head:table_base
    tail:table_ref* {
      tail.unshift(head);
      tail.forEach(tableInfo => {
        const { table, as } = tableInfo
        tableAlias[table] = table
        if (as) tableAlias[as] = table
        refreshColumnList(columnList)
      })
      return tail;
    }

table_ref
  = __ COMMA __ t:table_base { return t; }
  / __ t:table_join { return t; }

table_join
  = op:join_op __ t:table_base __ KW_USING __ LPAREN __ head:ident_without_kw_type tail:(__ COMMA __ ident_without_kw_type)* __ RPAREN {
      t.join = op;
      t.using = createList(head, tail);
      return t;
    }
  / op:join_op __ t:table_base __ expr:on_clause? {
      t.join = op;
      t.on   = expr;
      return t;
    }
  / op:(join_op / set_op) __ LPAREN __ stmt:union_stmt __ RPAREN __ alias:alias_clause? __ expr:on_clause? {
    stmt.parentheses = true;
    return {
      expr: stmt,
      as: alias,
      join: op,
      on: expr
    };
  }

//NOTE that, the table assigned to `var` shouldn't write in `table_join`
table_base
  = KW_DUAL {
      return {
        type: 'dual'
      };
  }
  / name:ident_name __ LPAREN __ l:expr_list __ RPAREN __ alias:alias_clause? {
      return {
        expr: {
          type: 'function',
          name: { name: [{ type: 'default', value: name }]},
          args: l,
        },
        as: alias,
      }
    }
  / t:table_name __ alias:alias_clause? {
      if (t.type === 'var') {
        t.as = alias;
        return t;
      } else {
        return {
          db: t.db,
          table: t.table,
          as: alias
        };
      }
    }
  / LPAREN __ stmt:union_stmt __ RPAREN __ alias:alias_clause? {
      stmt.parentheses = true;
      return {
        expr: stmt,
        as: alias
      };
    }

join_op
  = KW_LEFT __ KW_OUTER? __ KW_JOIN { return 'LEFT JOIN'; }
  / 'CROSS'i __ KW_JOIN { return 'CROSS JOIN'; }
  / (KW_INNER __)? KW_JOIN { return 'INNER JOIN'; }

table_name
  = dt:ident tail:(__ DOT __ ident)? {
      const obj = { db: null, table: dt };
      if (tail !== null) {
        obj.db = dt;
        obj.table = tail[3];
      }
      return obj;
    }
  / v:var_decl {
      v.db = null;
      v.table = v.name;
      return v;
    }
or_and_expr
	= head:expr tail:(__ (KW_AND / KW_OR) __ expr)* {
    const len = tail.length
    let result = head
    for (let i = 0; i < len; ++i) {
      result = createBinaryExpr(tail[i][1], result, tail[i][3])
    }
    return result
  }
on_clause
  = KW_ON __ e:or_and_where_expr { return e; }

where_clause
  = KW_WHERE __ e:or_and_where_expr { return e; }

group_by_clause
  = KW_GROUP __ KW_BY __ e:expr_list {
    return {
      columns: e.value
    }
  }

column_ref_index
  = column_ref_list / literal_list

column_ref_list
  = head:column_ref tail:(__ COMMA __ column_ref)* {
      return createList(head, tail);
    }

having_clause
  = KW_HAVING __ e:or_and_where_expr { return e; }

order_by_clause
  = KW_ORDER __ KW_BY __ l:order_by_list { return l; }

order_by_list
  = head:order_by_element tail:(__ COMMA __ order_by_element)* {
      return createList(head, tail);
    }

order_by_element
  = e:expr __ d:(KW_DESC / KW_ASC)? {
    const obj = { expr: e, type: d };
    return obj;
  }

number_or_param
  = literal_numeric
  / param

limit_clause
  = KW_LIMIT __ i1:(number_or_param) __ tail:((COMMA / KW_OFFSET) __ number_or_param)? {
      const res = [i1];
      if (tail) res.push(tail[2]);
      return {
        seperator: tail && tail[0] && tail[0].toLowerCase() || '',
        value: res
      };
    }

value_clause
  = KW_VALUES __ l:value_list  { return { type: 'values', values: l } }

value_list
  = head:value_item tail:(__ COMMA __ value_item)* {
      return createList(head, tail);
    }

value_item
  = LPAREN __ l:expr_list  __ RPAREN {
      return l;
    }

expr_list
  = head:expr tail:(__ COMMA __ expr)* {
      const el = { type: 'expr_list' };
      el.value = createList(head, tail);
      return el;
    }

interval_expr
  = KW_INTERVAL                    __
    e:expr                       __
    u: interval_unit {
      return {
        type: 'interval',
        expr: e,
        unit: u.toLowerCase(),
      }
    }

case_expr
  = KW_CASE                         __
    condition_list:case_when_then_list  __
    otherwise:case_else?            __
    KW_END __ KW_CASE? {
      if (otherwise) condition_list.push(otherwise);
      return {
        type: 'case',
        expr: null,
        args: condition_list
      };
    }
  / KW_CASE                         __
    expr:expr                      __
    condition_list:case_when_then_list  __
    otherwise:case_else?            __
    KW_END __ KW_CASE? {
      if (otherwise) condition_list.push(otherwise);
      return {
        type: 'case',
        expr: expr,
        args: condition_list
      };
    }

case_when_then_list
  = head:case_when_then __ tail:(__ case_when_then)* {
    return createList(head, tail, 1)
  }

case_when_then
  = KW_WHEN __ condition:or_and_where_expr __ KW_THEN __ result:expr {
    return {
      type: 'when',
      cond: condition,
      result: result
    };
  }

case_else = KW_ELSE __ result:expr {
    return { type: 'else', result: result };
  }

/**
 * Borrowed from PL/SQL ,the priority of below list IS ORDER BY DESC
 * ---------------------------------------------------------------------------------------------------
 * | +, -                                                     | identity, negation                   |
 * | *, /                                                     | multiplication, division             |
 * | +, -                                                     | addition, subtraction, concatenation |
 * | =, <, >, <=, >=, <>, !=, IS, LIKE, BETWEEN, IN           | comparion                            |
 * | !, NOT                                                   | logical negation                     |
 * | AND                                                      | conjunction                          |
 * | OR                                                       | inclusion                            |
 * ---------------------------------------------------------------------------------------------------
 */

_expr
  = or_expr
  / unary_expr

expr
  = _expr / union_stmt

unary_expr
  = op:additive_operator tail: (__ primary)+ {
    return createUnaryExpr(op, tail[0][1]);
  }

binary_column_expr
  = head:expr tail:(__ (KW_AND / KW_OR / LOGIC_OPERATOR) __ expr)* {
    const ast = head.ast
    if (ast && ast.type === 'select') {
      if (!(head.parentheses_symbol || head.parentheses || head.ast.parentheses || head.ast.parentheses_symbol) || ast.columns.length !== 1 || ast.columns[0].expr.column === '*') throw new Error('invalid column clause with select statement')
    }
    if (!tail || tail.length === 0) return head
    const len = tail.length
    let result = tail[len - 1][3]
    for (let i = len - 1; i >= 0; i--) {
      const left = i === 0 ? head : tail[i - 1][3]
      result = createBinaryExpr(tail[i][1], left, result)
    }
    return result
  }

or_and_where_expr
	= head:expr tail:(__ (KW_AND / KW_OR / COMMA) __ expr)* {
    const len = tail.length
    let result = head;
    let seperator = ''
    for (let i = 0; i < len; ++i) {
      if (tail[i][1] === ',') {
        seperator = ','
        if (!Array.isArray(result)) result = [result]
        result.push(tail[i][3])
      } else {
        result = createBinaryExpr(tail[i][1], result, tail[i][3]);
      }
    }
    if (seperator === ',') {
      const el = { type: 'expr_list' }
      el.value = result
      return el
    }
    return result
  }

or_expr
  = head:and_expr tail:(___ KW_OR __ and_expr)* {
      return createBinaryExprChain(head, tail);
    }

and_expr
  = head:not_expr tail:(___ KW_AND __ not_expr)* {
      return createBinaryExprChain(head, tail);
    }

//here we should use `NOT` instead of `comparision_expr` to support chain-expr
not_expr
  = comparison_expr
  / exists_expr
  / (KW_NOT / "!" !"=") __ expr:not_expr {
      return createUnaryExpr('NOT', expr);
    }

comparison_expr
  = left:additive_expr __ rh:comparison_op_right? {
      if (rh === null) return left;
      else if (rh.type === 'arithmetic') return createBinaryExprChain(left, rh.tail);
      else return createBinaryExpr(rh.op, left, rh.right);
    }
  / literal_string
  / column_ref

exists_expr
  = op:exists_op __ LPAREN __ stmt:union_stmt __ RPAREN {
    stmt.parentheses = true;
    return createUnaryExpr(op, stmt);
  }

exists_op
  = nk:(KW_NOT __ KW_EXISTS) { return nk[0] + ' ' + nk[2]; }
  / KW_EXISTS

comparison_op_right
  = arithmetic_op_right
  / in_op_right
  / between_op_right
  / is_op_right
  / like_op_right
  / regexp_op_right

arithmetic_op_right
  = l:(__ arithmetic_comparison_operator __ additive_expr)+ {
      return { type: 'arithmetic', tail: l };
    }

arithmetic_comparison_operator
  = ">=" / ">" / "<=" / "<>" / "<" / "==" / "=" / "!="

is_op_right
  = KW_IS __ right:additive_expr {
      return { op: 'IS', right: right };
    }
  / (KW_IS __ KW_NOT) __ right:additive_expr {
      return { op: 'IS NOT', right: right };
  }

between_op_right
  = op:between_or_not_between_op __  begin:additive_expr __ KW_AND __ end:additive_expr {
      return {
        op: op,
        right: {
          type: 'expr_list',
          value: [begin, end]
        }
      };
    }

between_or_not_between_op
  = nk:(KW_NOT __ KW_BETWEEN) { return nk[0] + ' ' + nk[2]; }
  / KW_BETWEEN

like_op
  = nk:(KW_NOT __ KW_LIKE) { return nk[0] + ' ' + nk[2]; }
  / KW_LIKE

regexp_op
  = n: KW_NOT? __ k:(KW_REGEXP / KW_RLIKE) {
    return n ? `${n} ${k}` : k
  }

escape_op
  = kw:'ESCAPE'i __ c:literal_string {
    // => { type: 'ESCAPE'; value: literal_string }
    return {
      type: 'ESCAPE',
      value: c,
    }
  }

in_op
  = nk:(KW_NOT __ KW_IN) { return nk[0] + ' ' + nk[2]; }
  / KW_IN

regexp_op_right
  = op:regexp_op __ b:'BINARY'i? __ e:(func_call / literal_string / column_ref) {
    return  { op: b ? `${op} ${b}` :  op, right: e };
  }
  / 'glob'i __ e:literal_string {
    return { op: 'GLOB', right: e }
  }

like_op_right
  = op:like_op __ right:(comparison_expr / literal) __ es:escape_op? {
      if (es) right.escape = es
      return { op: op, right: right };
    }

in_op_right
  = op:in_op __ LPAREN  __ l:expr_list __ RPAREN {
      return { op: op, right: l };
    }
  / op:in_op __ e:(var_decl / literal_string / column_ref / func_call) {
      return { op: op, right: e };
    }

additive_expr
  = head: multiplicative_expr
    tail:(__ additive_operator  __ multiplicative_expr)* {
      if (tail && tail.length && head.type === 'column_ref' && head.column === '*') throw new Error(JSON.stringify({
        message: 'args could not be star column in additive expr',
        ...getLocationObject(),
      }))
      return createBinaryExprChain(head, tail);
    }

additive_operator
  = "+" / "-"

multiplicative_expr
  = head:unary_expr_or_primary
    tail:(__  (multiplicative_operator / LOGIC_OPERATOR)  __ unary_expr_or_primary)* {
      return createBinaryExprChain(head, tail)
    }

multiplicative_operator
  = "*" / "/" / "%" / "||" / '&' / '>>' / '<<' / '^' / '|'

primary
  = cast_expr
  / literal
  / interval_expr
  / aggr_func
  / func_call
  / case_expr
  / column_ref
  / param
  / LPAREN __ list:or_and_where_expr __ RPAREN {
        list.parentheses = true;
        return list;
    }
  / var_decl
  / __ prepared_symbol:'?' {
    return {
      type: 'origin',
      value: prepared_symbol
    }
  }

unary_expr_or_primary
  = jsonb_expr
  / op:(unary_operator) tail:(__ unary_expr_or_primary) {
    // if (op === '!') op = 'NOT'
    return createUnaryExpr(op, tail[1])
  }

unary_operator
  = '!' / '-' / '+' / '~'

jsonb_expr
  = head:primary __ tail: (__ ('?|' / '?&' / '?' / '#-' / '#>>' / '#>' / DOUBLE_ARROW / SINGLE_ARROW / '@>' / '<@') __  primary)* {
    // => primary | binary_expr
    if (!tail || tail.length === 0) return head
    return createBinaryExprChain(head, tail)
  }

column_ref
  = tbl:ident __ DOT __ col:column_without_kw ce:(__ collate_expr)? {
      columnList.add(`select::${tbl}::${col}`);
      return {
        type: 'column_ref',
        table: tbl,
        column: col,
        collate: ce && ce[1],
      };
    }
  / col:column ce:(__ collate_expr)? {
      columnList.add(`select::null::${col}`);
      return {
        type: 'column_ref',
        table: null,
        column: col,
        collate: ce && ce[1],
      };
    }

ident_without_kw_type
  = n:ident_name {
    return { type: 'default', value: n }
  }
  / quoted_ident_type

ident
  = name:ident_name !{ return reservedMap[name.toUpperCase()] === true; } {
      return name;
    }
  / name:quoted_ident {
      return name;
    }

alias_ident
  = name:ident_name !{
      if (reservedMap[name.toUpperCase()] === true) throw new Error("Error: "+ JSON.stringify(name)+" is a reserved word, can not as alias clause");
      return false
    } {
      return name;
    }
  / name:quoted_ident {
      return name;
    }

quoted_ident_type
  = double_quoted_ident / single_quoted_ident / backticks_quoted_ident

quoted_ident
  = v:(double_quoted_ident / single_quoted_ident / backticks_quoted_ident) {
    return v.value
  }

double_quoted_ident
  = '"' chars:[^"]+ '"' {
    return {
      type: 'double_quote_string',
      value: chars.join('')
    }
  }

single_quoted_ident
  = "'" chars:[^']+ "'" {
    return {
      type: 'single_quote_string',
      value: chars.join('')
    }
  }

backticks_quoted_ident
  = "`" chars:[^`]+ "`" {
    return {
      type: 'backticks_quote_string',
      value: chars.join('')
    }
  }

column_without_kw
  = name:column_name {
    return name;
  }
  / quoted_ident

column
  = name:column_name !{ return reservedMap[name.toUpperCase()] === true; } { return name; }
  / quoted_ident

column_name
  =  start:ident_start parts:column_part* { return start + parts.join(''); }

ident_name
  =  start:ident_start parts:ident_part* { return start + parts.join(''); }

ident_start = [A-Za-z_]

ident_part  = [A-Za-z0-9_]

// to support column name like `cf1:name` in hbase
column_part  = [A-Za-z0-9_:\u4e00-\u9fa5\u00C0-\u017F]

param
  = l:(':' ident_name) {
      return { type: 'param', value: l[1] };
    }

aggr_func
  = aggr_fun_count
  / aggr_fun_smma

aggr_fun_smma
  = name:KW_SUM_MAX_MIN_AVG  __ LPAREN __ e:additive_expr __ RPAREN __ bc:over_partition?  {
      return {
        type: 'aggr_func',
        name: name,
        args: {
          expr: e
        },
        over: bc,
        ...getLocationObject(),
      };
    }

KW_SUM_MAX_MIN_AVG
  = KW_SUM / KW_MAX / KW_MIN / KW_AVG

on_update_current_timestamp
  = KW_ON __ KW_UPDATE __ kw:KW_CURRENT_TIMESTAMP __ LPAREN __ l:expr_list? __ RPAREN{
    return {
      type: 'on update',
      keyword: kw,
      parentheses: true,
      expr: l
    }
  }
  / KW_ON __ KW_UPDATE __ kw:KW_CURRENT_TIMESTAMP {
    return {
      type: 'on update',
      keyword: kw,
    }
  }

over_partition
  = KW_OVER __ LPAREN __ p:partition_by_clause? __ l:order_by_clause? __ RPAREN {
    return {
      partitionby: p,
      orderby: l
    }
  }
  / on_update_current_timestamp

partition_by_clause
  = KW_PARTITION __ KW_BY __ bc:column_clause { return bc; }
aggr_fun_count
  = name:(KW_COUNT / KW_GROUP_CONCAT) __ LPAREN __ arg:count_arg __ RPAREN __ bc:over_partition? {
      return {
        type: 'aggr_func',
        name: name,
        args: arg,
        over: bc
      };
    }

count_arg
  = e:star_expr { return { expr: e }; }
  / d:KW_DISTINCT? __ LPAREN __ c:expr __ RPAREN tail:(__ (KW_AND / KW_OR) __ expr)* __ or:order_by_clause? {
    const len = tail.length
    let result = c
    result.parentheses = true
    for (let i = 0; i < len; ++i) {
      result = createBinaryExpr(tail[i][1], result, tail[i][3])
    }
    return {
      distinct: d,
      expr: result,
      orderby: or,
    };
  }
  / d:KW_DISTINCT? __ c:or_and_expr __ or:order_by_clause? { return { distinct: d, expr: c, orderby: or }; }

star_expr
  = "*" { return { type: 'star', value: '*' }; }

func_call
  = name:scalar_func __ LPAREN __ l:expr_list? __ RPAREN __ bc:over_partition? {
      return {
        type: 'function',
        name: { name: [{ type: 'default', value: name }] },
        args: l ? l: { type: 'expr_list', value: [] },
        over: bc,
        ...getLocationObject(),
      };
    }
  / f:scalar_time_func __ up:on_update_current_timestamp? {
    return {
        type: 'function',
        name: { name: [{ type: 'origin', value: f }] },
        over: up,
        ...getLocationObject(),
    }
  }
  / name:proc_func_name __ LPAREN __ l:or_and_where_expr? __ RPAREN __ bc:over_partition? {
    if (l && l.type !== 'expr_list') l = { type: 'expr_list', value: [l] }
      return {
        type: 'function',
        name: name,
        args: l ? l: { type: 'expr_list', value: [] },
        over: bc,
        ...getLocationObject(),
      };
    }
scalar_time_func
  = KW_CURRENT_DATE
  / KW_CURRENT_TIME
  / KW_CURRENT_TIMESTAMP
scalar_func
  = scalar_time_func
  / KW_CURRENT_USER
  / KW_USER
  / KW_SESSION_USER
  / KW_SYSTEM_USER

cast_expr
  = c:KW_CAST __ LPAREN __ e:expr __ KW_AS __ t:data_type __ RPAREN {
    return {
      type: 'cast',
      keyword: c.toLowerCase(),
      expr: e,
      symbol: 'as',
      target: [t]
    };
  }
  / c:KW_CAST __ LPAREN __ e:expr __ KW_AS __ KW_DECIMAL __ LPAREN __ precision:int __ RPAREN __ RPAREN {
    return {
      type: 'cast',
      keyword: c.toLowerCase(),
      expr: e,
      symbol: 'as',
      target: [{
        dataType: 'DECIMAL(' + precision + ')'
      }]
    };
  }
  / c:KW_CAST __ LPAREN __ e:expr __ KW_AS __ KW_DECIMAL __ LPAREN __ precision:int __ COMMA __ scale:int __ RPAREN __ RPAREN {
      return {
        type: 'cast',
        keyword: c.toLowerCase(),
        expr: e,
        symbol: 'as',
        target: [{
          dataType: 'DECIMAL(' + precision + ', ' + scale + ')'
        }]
      };
    }
  / c:KW_CAST __ LPAREN __ e:expr __ KW_AS __ s:signedness __ t:KW_INTEGER? __ RPAREN { /* MySQL cast to un-/signed integer */
    return {
      type: 'cast',
      keyword: c.toLowerCase(),
      expr: e,
      symbol: 'as',
      target: [{
        dataType: s + (t ? ' ' + t: '')
      }]
    };
  }

signedness
  = KW_SIGNED
  / KW_UNSIGNED

literal
  = b:'BINARY'i? __ s:literal_string ca:(__ collate_expr)? {
    if (b) s.prefix = b.toLowerCase()
    if (ca) s.suffix = { collate: ca[1] }
    return s
  }
  / literal_numeric
  / literal_bool
  / literal_null
  / literal_datetime

literal_list
  = head:literal tail:(__ COMMA __ literal)* {
      return createList(head, tail);
    }

literal_null
  = KW_NULL {
      return { type: 'null', value: null };
    }

literal_bool
  = KW_TRUE {
      return { type: 'bool', value: true };
    }
  / KW_FALSE {
      return { type: 'bool', value: false };
    }

literal_string
  = b:'_binary'i ? __ r:'X'i ca:("'" [0-9A-Fa-f]* "'") {
      return {
        type: 'hex_string',
        prefix: b,
        value: ca[1].join('')
      };
    }
  / b:'_binary'i ? __ r:'b'i ca:("'" [0-9A-Fa-f]* "'") {
      return {
        type: 'bit_string',
        prefix: b,
        value: ca[1].join('')
      };
    }
  / b:'_binary'i ? __ r:'0x' ca:([0-9A-Fa-f]*) {
    return {
        type: 'full_hex_string',
        prefix: b,
        value: ca.join('')
      };
  }
  / ca:("'" single_char* "'") __ !(DOT / LPAREN) {
      return {
        type: 'single_quote_string',
        value: ca[1].join('')
      };
    }
  / ca:("\"" single_quote_char* "\"") __ !(DOT / LPAREN) {
      return {
        type: 'double_quote_string',
        value: ca[1].join('')
      };
    }

literal_datetime
  = type:(KW_TIME / KW_DATE / KW_TIMESTAMP / KW_DATETIME) __ ca:("'" single_char* "'") {
      return {
        type: type.toLowerCase(),
        value: ca[1].join('')
      };
    }
  / type:(KW_TIME / KW_DATE / KW_TIMESTAMP / KW_DATETIME) __ ca:("\"" single_quote_char* "\"") {
      return {
        type: type.toLowerCase(),
        value: ca[1].join('')
      };
    }

single_quote_char
  = [^"\\\0-\x1F\x7f]
  / escape_char

single_char
  = [^'] // remove \0-\x1F\x7f pnCtrl char [^'\\\0-\x1F\x7f]
  / escape_char

escape_char
  = "''"

literal_numeric
  = n:number {
      if (n && n.type === 'bigint') return n
      return { type: 'number', value: n };
    }

number
  = int_:int frac:frac exp:exp {
    const numStr = int_ + frac + exp
    return {
      type: 'bigint',
      value: numStr
    }
  }
  / int_:int frac:frac {
    const numStr = int_ + frac
    if (isBigInt(int_)) return {
      type: 'bigint',
      value: numStr
    }
    const fixed = frac.length >= 1 ? frac.length - 1 : 0
    return parseFloat(numStr).toFixed(fixed);
  }
  / int_:int exp:exp {
    const numStr = int_ + exp
    return {
      type: 'bigint',
      value: numStr
    }
  }
  / int_:int {
    if (isBigInt(int_)) return {
      type: 'bigint',
      value: int_
    }
    return parseFloat(int_);
  }

int
  = digits
  / digit:digit
  / op:("-" / "+" ) digits:digits { return op + digits; }
  / op:("-" / "+" ) digit:digit { return op + digit; }

frac
  = "." digits:digits? {
    if (!digits) return ''
    return "." + digits;
  }

exp
  = e:e digits:digits { return e + digits; }

digits
  = digits:digit+ { return digits.join(""); }

digit   = [0-9]

e
  = e:[eE] sign:[+-]? { return e + (sign !== null ? sign: ''); }

KW_NULL     = "NULL"i       !ident_start
KW_TRUE     = "TRUE"i       !ident_start
KW_FALSE    = "FALSE"i      !ident_start

KW_SELECT   = "SELECT"i     !ident_start
KW_UPDATE   = "UPDATE"i     !ident_start
KW_RECURSIVE= "RECURSIVE"i   !ident_start
KW_PARTITION = "PARTITION"i !ident_start { return 'PARTITION' }

KW_FROM     = "FROM"i       !ident_start
KW_AS       = "AS"i         !ident_start
KW_COLLATE  = "COLLATE"i    !ident_start { return 'COLLATE'; }

KW_ON       = "ON"i       !ident_start
KW_LEFT     = "LEFT"i     !ident_start
KW_INNER    = "INNER"i    !ident_start
KW_JOIN     = "JOIN"i     !ident_start
KW_OUTER    = "OUTER"i    !ident_start
KW_OVER     = "OVER"i     !ident_start
KW_UNION    = "UNION"i    !ident_start
KW_VALUES   = "VALUES"i   !ident_start
KW_USING    = "USING"i    !ident_start

KW_WHERE    = "WHERE"i      !ident_start
KW_WITH     = "WITH"i       !ident_start

KW_GROUP    = "GROUP"i      !ident_start
KW_BY       = "BY"i         !ident_start
KW_ORDER    = "ORDER"i      !ident_start
KW_HAVING   = "HAVING"i     !ident_start

KW_LIMIT    = "LIMIT"i      !ident_start
KW_OFFSET   = "OFFSET"i     !ident_start { return 'OFFSET'; }

KW_ASC      = "ASC"i        !ident_start { return 'ASC'; }
KW_DESC     = "DESC"i       !ident_start { return 'DESC'; }
KW_ALL      = "ALL"i        !ident_start { return 'ALL'; }
KW_DISTINCT = "DISTINCT"i   !ident_start { return 'DISTINCT';}

KW_BETWEEN  = "BETWEEN"i    !ident_start { return 'BETWEEN'; }
KW_IN       = "IN"i         !ident_start { return 'IN'; }
KW_IS       = "IS"i         !ident_start { return 'IS'; }
KW_LIKE     = "LIKE"i       !ident_start { return 'LIKE'; }
KW_RLIKE    = "RLIKE"i      !ident_start { return 'RLIKE'; }
KW_REGEXP   = "REGEXP"i     !ident_start { return 'REGEXP'; }
KW_EXISTS   = "EXISTS"i     !ident_start { return 'EXISTS'; }

KW_NOT      = "NOT"i        !ident_start { return 'NOT'; }
KW_AND      = "AND"i        !ident_start { return 'AND'; }
KW_OR       = "OR"i         !ident_start { return 'OR'; }

KW_COUNT    = "COUNT"i      !ident_start { return 'COUNT'; }
KW_GROUP_CONCAT = "GROUP_CONCAT"i  !ident_start { return 'GROUP_CONCAT'; }
KW_MAX      = "MAX"i        !ident_start { return 'MAX'; }
KW_MIN      = "MIN"i        !ident_start { return 'MIN'; }
KW_SUM      = "SUM"i        !ident_start { return 'SUM'; }
KW_AVG      = "AVG"i        !ident_start { return 'AVG'; }

KW_CASE     = "CASE"i       !ident_start
KW_WHEN     = "WHEN"i       !ident_start
KW_THEN     = "THEN"i       !ident_start
KW_ELSE     = "ELSE"i       !ident_start
KW_END      = "END"i        !ident_start

KW_CAST     = "CAST"i       !ident_start { return 'CAST' }

KW_BIT      = "BIT"i      !ident_start { return 'BIT'; }
KW_CHAR     = "CHAR"i     !ident_start { return 'CHAR'; }
KW_VARCHAR  = "VARCHAR"i  !ident_start { return 'VARCHAR';}
KW_NUMERIC  = "NUMERIC"i  !ident_start { return 'NUMERIC'; }
KW_DECIMAL  = "DECIMAL"i  !ident_start { return 'DECIMAL'; }
KW_SIGNED   = "SIGNED"i   !ident_start { return 'SIGNED'; }
KW_UNSIGNED = "UNSIGNED"i !ident_start { return 'UNSIGNED'; }
KW_INT      = "INT"i      !ident_start { return 'INT'; }
KW_ZEROFILL = "ZEROFILL"i !ident_start { return 'ZEROFILL'; }
KW_INTEGER  = "INTEGER"i  !ident_start { return 'INTEGER'; }
KW_JSON     = "JSON"i     !ident_start { return 'JSON'; }
KW_SMALLINT = "SMALLINT"i !ident_start { return 'SMALLINT'; }
KW_TINYINT  = "TINYINT"i  !ident_start { return 'TINYINT'; }
KW_TINYTEXT = "TINYTEXT"i !ident_start { return 'TINYTEXT'; }
KW_TEXT     = "TEXT"i     !ident_start { return 'TEXT'; }
KW_MEDIUMTEXT = "MEDIUMTEXT"i  !ident_start { return 'MEDIUMTEXT'; }
KW_LONGTEXT  = "LONGTEXT"i  !ident_start { return 'LONGTEXT'; }
KW_BIGINT   = "BIGINT"i   !ident_start { return 'BIGINT'; }
KW_ENUM     = "ENUM"i   !ident_start { return 'ENUM'; }
KW_FLOAT   = "FLOAT"i   !ident_start { return 'FLOAT'; }
KW_DOUBLE   = "DOUBLE"i   !ident_start { return 'DOUBLE'; }
KW_REAL     = "REAL"i   !ident_start { return 'REAL'; }
KW_DATE     = "DATE"i     !ident_start { return 'DATE'; }
KW_DATETIME     = "DATETIME"i     !ident_start { return 'DATETIME'; }
KW_TIME     = "TIME"i     !ident_start { return 'TIME'; }
KW_TIMESTAMP= "TIMESTAMP"i!ident_start { return 'TIMESTAMP'; }
KW_USER     = "USER"i     !ident_start { return 'USER'; }

KW_CURRENT_DATE     = "CURRENT_DATE"i !ident_start { return 'CURRENT_DATE'; }
KW_INTERVAL         = "INTERVAL"i !ident_start { return 'INTERVAL'; }
KW_UNIT_YEAR        = "YEAR"i !ident_start { return 'YEAR'; }
KW_UNIT_MONTH       = "MONTH"i !ident_start { return 'MONTH'; }
KW_UNIT_DAY         = "DAY"i !ident_start { return 'DAY'; }
KW_UNIT_HOUR        = "HOUR"i !ident_start { return 'HOUR'; }
KW_UNIT_MINUTE      = "MINUTE"i !ident_start { return 'MINUTE'; }
KW_UNIT_SECOND      = "SECOND"i !ident_start { return 'SECOND'; }
KW_CURRENT_TIME     = "CURRENT_TIME"i !ident_start { return 'CURRENT_TIME'; }
KW_CURRENT_TIMESTAMP= "CURRENT_TIMESTAMP"i !ident_start { return 'CURRENT_TIMESTAMP'; }
KW_CURRENT_USER     = "CURRENT_USER"i !ident_start { return 'CURRENT_USER'; }
KW_SESSION_USER     = "SESSION_USER"i !ident_start { return 'SESSION_USER'; }
KW_SYSTEM_USER      = "SYSTEM_USER"i !ident_start { return 'SYSTEM_USER'; }

KW_VAR__PRE_AT = '@'
KW_VAR__PRE_AT_AT = '@@'
KW_VAR_PRE_DOLLAR = '$'
KW_VAR_PRE
  = KW_VAR__PRE_AT_AT / KW_VAR__PRE_AT / KW_VAR_PRE_DOLLAR
KW_ASSIGIN_EQUAL = '='

KW_DUAL = "DUAL"i

// MySQL Alter
OPT_SQL_CALC_FOUND_ROWS = "SQL_CALC_FOUND_ROWS"i
OPT_SQL_CACHE           = "SQL_CACHE"i
OPT_SQL_NO_CACHE        = "SQL_NO_CACHE"i
OPT_SQL_SMALL_RESULT    = "SQL_SMALL_RESULT"i
OPT_SQL_BIG_RESULT      = "SQL_BIG_RESULT"i
OPT_SQL_BUFFER_RESULT   = "SQL_BUFFER_RESULT"i

//special character
DOT       = '.'
COMMA     = ','
STAR      = '*'
LPAREN    = '('
RPAREN    = ')'

SEMICOLON = ';'
SINGLE_ARROW = '->'
DOUBLE_ARROW = '->>'

OPERATOR_CONCATENATION = '||'
OPERATOR_AND = '&&'
LOGIC_OPERATOR = OPERATOR_CONCATENATION / OPERATOR_AND

// separator
__
  = (whitespace / comment)*

___
  = (whitespace / comment)+

comment
  = block_comment
  / line_comment
  / pound_sign_comment

block_comment
  = "/*" (!"*/" char)* "*/"

line_comment
  = "--" (!EOL char)*

pound_sign_comment
  = "#" (!EOL char)*

char = .

interval_unit
  = KW_UNIT_YEAR
  / KW_UNIT_MONTH
  / KW_UNIT_DAY
  / KW_UNIT_HOUR
  / KW_UNIT_MINUTE
  / KW_UNIT_SECOND

whitespace =
  [ \t\n\r]

EOL
  = EOF
  / [\n\r]+

EOF = !.

//begin procedure extension
proc_func_name
  = dt:ident_without_kw_type tail:(__ DOT __ ident_without_kw_type)? {
      const result = { name: [dt] }
      if (tail !== null) {
        result.schema = dt
        result.name = [tail[3]]
      }
      return result
    }

var_decl
  = p: KW_VAR_PRE d: without_prefix_var_decl {
    //push for analysis
    return {
      type: 'var',
      ...d,
      prefix: p
    };
  }

without_prefix_var_decl
  = name:ident_name m:mem_chain {
    return {
      type: 'var',
      name: name,
      members: m,
      prefix: null,
    };
  }
  / n:literal_numeric {
    return {
      type: 'var',
      name: n.value,
      members: [],
      quoted: null,
      prefix: null,
    }
  }

mem_chain
  = l:('.' ident_name)* {
    const s = [];
    for (let i = 0; i < l.length; i++) {
      s.push(l[i][1]);
    }
    return s;
  }

data_type
  = character_string_type
  / numeric_type
  / datetime_type
  / json_type
  / text_type
  / enum_type
  / boolean_type
  / blob_type

blob_type
  = b:('blob'i / 'tinyblob'i / 'mediumblob'i / 'longblob'i) { return { dataType: b.toUpperCase() }; }

boolean_type
  = 'boolean'i { return { dataType: 'BOOLEAN' }; }

character_string_type
  = t:(KW_CHAR / KW_VARCHAR) __ LPAREN __ l:[0-9]+ __ RPAREN {
    return { dataType: t, length: parseInt(l.join(''), 10), parentheses: true };
  }
  / t:KW_CHAR { return { dataType: t }; }
  / t:KW_VARCHAR { return { dataType: t }; }

numeric_type_suffix
  = un: KW_UNSIGNED? __ ze: KW_ZEROFILL? {
    const result = []
    if (un) result.push(un)
    if (ze) result.push(ze)
    return result
  }
numeric_type
  = t:(KW_NUMERIC / KW_DECIMAL / KW_INT / KW_INTEGER / KW_SMALLINT / KW_TINYINT / KW_BIGINT / KW_FLOAT / KW_DOUBLE / KW_BIT / KW_REAL) __ LPAREN __ l:[0-9]+ __ r:(COMMA __ [0-9]+)? __ RPAREN __ s:numeric_type_suffix? { return { dataType: t, length: parseInt(l.join(''), 10), scale: r && parseInt(r[2].join(''), 10), parentheses: true, suffix: s }; }
  / t:(KW_NUMERIC / KW_DECIMAL / KW_INT / KW_INTEGER / KW_SMALLINT / KW_TINYINT / KW_BIGINT / KW_FLOAT / KW_DOUBLE / KW_REAL)l:[0-9]+ __ s:numeric_type_suffix? { return { dataType: t, length: parseInt(l.join(''), 10), suffix: s }; }
  / t:(KW_NUMERIC / KW_DECIMAL / KW_INT / KW_INTEGER / KW_SMALLINT / KW_TINYINT / KW_BIGINT / KW_FLOAT / KW_DOUBLE / KW_REAL) __ s:numeric_type_suffix? __{ return { dataType: t, suffix: s }; }

datetime_type
  = t:(KW_DATE / KW_DATETIME / KW_TIME / KW_TIMESTAMP) __ LPAREN __ l:[0-6] __ RPAREN __ s:numeric_type_suffix? { return { dataType: t, length: parseInt(l, 10), parentheses: true }; }
  / t:(KW_DATE / KW_DATETIME / KW_TIME / KW_TIMESTAMP) { return { dataType: t }; }

enum_type
  = t:KW_ENUM __ e:value_item {
    e.parentheses = true
    return {
      dataType: t,
      expr: e
    }
  }

json_type
  = t:KW_JSON { return { dataType: t }; }

text_type
  = t:(KW_TINYTEXT / KW_TEXT / KW_MEDIUMTEXT / KW_LONGTEXT) { return { dataType: t }}
