// Apla Software includes an integrated development
// environment with a multi-level system for the management
// of access rights to data, interfaces, and Smart contracts. The
// technical characteristics of the Apla Software are indicated in
// Apla Technical Paper.
//
// Apla Users are granted a permission to deal in the Apla
// Software without restrictions, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of Apla Software, and to permit persons
// to whom Apla Software is furnished to do so, subject to the
// following conditions:
// * the copyright notice of GenesisKernel and EGAAS S.A.
// and this permission notice shall be included in all copies or
// substantial portions of the software;
// * a result of the dealing in Apla Software cannot be
// implemented outside of the Apla Platform environment.
//
// THE APLA SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY
// OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED
// TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A
// PARTICULAR PURPOSE, ERROR FREE AND NONINFRINGEMENT. IN
// NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR
// THE USE OR OTHER DEALINGS IN THE APLA SOFTWARE.

package vde

var systemParametersDataSQL = `
INSERT INTO "1_system_parameters" ("id","name", "value", "conditions") VALUES 
	('1','default_ecosystem_page', '', 'true'),
	('2','default_ecosystem_menu', '', 'true'),
	('3','default_ecosystem_contract', '', 'true'),
	('4','gap_between_blocks', '2', 'true'),
	('5','rollback_blocks', '60', 'true'),
	('8','full_nodes', '', 'true'),
	('9','number_of_nodes', '101', 'true'),
	('11','price_create_contract', '200', 'true'),
	('14','price_create_menu', '100', 'true'),
	('15','price_create_page', '100', 'true'),
	('16','blockchain_url', '', 'true'),
	('17','max_block_size', '67108864', 'true'),
	('18','max_tx_size', '33554432', 'true'),
	('19','max_tx_block', '1000', 'true'),
	('20','max_columns', '50', 'true'),
	('21','max_indexes', '5', 'true'),
	('22','max_tx_block_per_user', '100', 'true'),
	('23','max_fuel_tx', '20000', 'true'),
	('24','max_fuel_block', '100000', 'true'),
	('25','commission_size', '3', 'true'),
	('26','commission_wallet', '', 'true'),
	('27','fuel_rate', '[["1","100000000000"]]', 'true'),
	('28','price_exec_address_to_id', '10', 'true'),
	('29','price_exec_id_to_address', '10', 'true'),
	('31','price_exec_sha256', '50', 'true'),
	('32','price_exec_pub_to_id', '10', 'true'),
	('33','price_exec_ecosys_param', '10', 'true'),
	('34','price_exec_sys_param_string', '10', 'true'),
	('35','price_exec_sys_param_int', '10', 'true'),
	('36','price_exec_sys_fuel', '10', 'true'),
	('37','price_exec_validate_condition', '30', 'true'),
	('38','price_exec_eval_condition', '20', 'true'),
	('39','price_exec_has_prefix', '10', 'true'),
	('40','price_exec_contains', '10', 'true'),
	('41','price_exec_replace', '10', 'true'),
	('42','price_exec_join', '10', 'true'),
	('43','price_exec_update_lang', '10', 'true'),
	('44','price_exec_size', '10', 'true'),
	('45','price_exec_substr', '10', 'true'),
	('46','price_exec_contracts_list', '10', 'true'),
	('47','price_exec_is_object', '10', 'true'),
	('48','price_exec_compile_contract', '100', 'true'),
	('49','price_exec_flush_contract', '50', 'true'),
	('50','price_exec_eval', '10', 'true'),
	('51','price_exec_len', '5', 'true'),
	('52','price_exec_bind_wallet', '10', 'true'),
	('53','price_exec_unbind_wallet', '10', 'true'),
	('54','price_exec_create_ecosystem', '100', 'true'),
	('55','price_exec_table_conditions', '100', 'true'),
	('56','price_exec_create_table', '100', 'true'),
	('57','price_exec_perm_table', '100', 'true'),
	('58','price_exec_column_condition', '50', 'true'),
	('59','price_exec_create_column', '50', 'true'),
	('60','price_exec_perm_column', '50', 'true'),
	('61','price_exec_json_to_map', '50', 'true'),
	('62','max_block_generation_time', '2000', 'true'),
	('63','block_reward','1000','true'),
	('64','incorrect_blocks_per_day','10','true'),
	('65','node_ban_time','86400000','true'),
	('66','node_ban_time_local','1800000','true'),
	('67','max_forsign_size','1000000','true'),
	('68','price_tx_data','0','true'),
	('69','price_exec_contract_by_name', '0', 'true'),
	('70','price_exec_contract_by_id', '0', 'true');
`
